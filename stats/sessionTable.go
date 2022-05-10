//Package tsyncmap : a timeout based syncmap
package stats

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"reflect"
	"sync"
	"sync/atomic"
	"time"
)

// Map is the base structure for tsyncmap.
type SessionTable struct {
	Name        string
	Data        sync.Map
	Timeout     int64
	CheckFreq   int64
	ExpireTime  sync.Map
	Verbose     bool
	TimeOutChan chan *Metric
	stopSignal  *int32 //atomic Counters,stop when cnt =1

}

//NewMap is a construct function to create tsyncmap.
func NewSessionTable(timeout time.Duration, checkfreq time.Duration, verbose bool) *SessionTable {
	t := int64(timeout)
	f := int64(checkfreq)

	r := &SessionTable{
		Name:        "traceroute",
		Timeout:     t,
		CheckFreq:   f,
		Verbose:     verbose,
		TimeOutChan: make(chan *Metric, 100),
	}

	var sig int32 = 0
	r.stopSignal = &sig
	atomic.StoreInt32(r.stopSignal, 0)

	go r.Run()
	return r
}

//Load returns the value from tsyncmap
func (tmap *SessionTable) Load(key interface{}) (value interface{}, ok bool) {
	return tmap.Data.Load(key)
}

type keyjson struct {
	Key string
}

func (tmap *SessionTable) LoadRestApi(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	v := new(keyjson)
	_ = json.NewDecoder(r.Body).Decode(v)

	if v.Key == "internal_fetch_keylist" {
		var result []interface{}
		tmap.Data.Range(func(k, v interface{}) bool {
			result = append(result, k)
			return true
		})
		json.NewEncoder(w).Encode(result)
	} else {
		result, find := tmap.Data.Load(v.Key)
		if find {
			json.NewEncoder(w).Encode(result)
		} else {
			json.NewEncoder(w).Encode("")

		}
	}
}

func (tmap *SessionTable) GetRemainTime(key interface{}) (time.Duration, error) {
	exp, ok := tmap.ExpireTime.Load(key)
	if ok {
		remainTime := exp.(time.Time).Sub(time.Now())
		return remainTime, nil
	}
	return 0 * time.Second, fmt.Errorf("key does not exist")
}

//Store is used save the key,value pairs in tsyncmap
func (tmap *SessionTable) Store(key interface{}, value interface{}, currentTime time.Time) {
	//Check ExpireTime Map.
	exp, ok := tmap.ExpireTime.Load(key)
	if !ok {
		expireTime := currentTime.Add(time.Duration(tmap.Timeout))
		tmap.ExpireTime.Store(key, expireTime)
	} else {
		elapsed := exp.(time.Time).Sub(currentTime)
		//elapsed time less than half of timeout, update ExpireTime Store.
		if elapsed < time.Duration(tmap.Timeout/2) {
			expireTime := currentTime.Add(time.Duration(tmap.Timeout))
			tmap.ExpireTime.Store(key, expireTime)
		}
	}
	tmap.Data.Store(key, value)
}

//UpdateTime is used update specific key's expiretime.
func (tmap *SessionTable) UpdateTime(key interface{}, currentTime time.Time) {
	expireTime := currentTime.Add(time.Duration(tmap.Timeout))
	tmap.ExpireTime.Store(key, expireTime)
}

func (tmap *SessionTable) Delete(key interface{}) {
	tmap.Data.Delete(key)
	tmap.ExpireTime.Delete(key)
}

//Run is a coroutine to help tsyncmap manage the expire data.
func (tmap *SessionTable) Run() {
	atomic.StoreInt32(tmap.stopSignal, 0)
	rand.Seed(time.Now().UnixNano())

	r := tmap.CheckFreq / 5
	for {
		currentTime := time.Now()
		tmap.ExpireTime.Range(func(k, v interface{}) bool {
			value := v.(time.Time)
			if value.Sub(currentTime) < 0 {
				//fmt.Println("DEBUG:::DELETE-KEY", reflect.ValueOf(k))
				tmpData, _ := tmap.Data.Load(k)
				tmap.TimeOutChan <- tmpData.(*Metric)
				tmap.Data.Delete(k)
				tmap.ExpireTime.Delete(k)
			}
			return true
		})
		if tmap.Verbose {
			tmap.ShowExpireTime()
			tmap.ShowData()
		}
		time.Sleep(time.Duration(tmap.CheckFreq + rand.Int63n(r)))
		if atomic.LoadInt32(tmap.stopSignal) == 1 {
			break
		}
	}
}

func (tmap *SessionTable) Stop() {
	atomic.StoreInt32(tmap.stopSignal, 1)
}

func (tmap *SessionTable) ShowExpireTime() {
	fmt.Printf("%10s:--------------------Expire Time Table-------------------------------\n", tmap.Name)
	i := 1
	tmap.ExpireTime.Range(func(k, v interface{}) bool {
		value := v.(time.Time)
		key := reflect.ValueOf(k)
		fmt.Printf("%10s:[%6d]Key:%v | ExipreTime: %v \n", tmap.Name, i, key, value)
		i++
		return true
	})
	fmt.Printf("%10s:--------------------------------------------------------------------\n\n", tmap.Name)
}

func (tmap *SessionTable) ShowData() {
	fmt.Printf("%10s:---------------------Data Table-------------------------------------\n", tmap.Name)
	i := 1
	tmap.Data.Range(func(k, v interface{}) bool {
		value := reflect.ValueOf(v)
		key := reflect.ValueOf(k)
		fmt.Printf("%10s:[%6d]Key:%v | Value: %v \n", tmap.Name, i, key, value)
		i++
		return true
	})
	fmt.Printf("%10s:--------------------------------------------------------------------\n\n", tmap.Name)
}
