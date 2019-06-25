//   Copyright (C) 2018 ZVChain
//
//   This program is free software: you can redistribute it and/or modify
//   it under the terms of the GNU General Public License as published by
//   the Free Software Foundation, either version 3 of the License, or
//   (at your option) any later version.
//
//   This program is distributed in the hope that it will be useful,
//   but WITHOUT ANY WARRANTY; without even the implied warranty of
//   MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
//   GNU General Public License for more details.
//
//   You should have received a copy of the GNU General Public License
//   along with this program.  If not, see <https://www.gnu.org/licenses/>.

package common

import (
	"os"
	"strings"
	"sync"

	ini "github.com/glacjay/goini"
)

/*
**  Creator: pxf
**  Date: 2018/4/10 下午2:13
**  Description:
 */

type ConfManager interface {
	//read basic conf from tas.conf file
	//返回section组下的key的值, 若未配置, 则返回默认值defv
	GetString(section string, key string, defaultValue string) string
	GetBool(section string, key string, defaultValue bool) bool
	GetDouble(section string, key string, defaultValue float64) float64
	GetInt(section string, key string, defaultValue int) int

	//set basic conf to tas.conf file
	SetString(section string, key string, value string)
	SetBool(section string, key string, value bool)
	SetDouble(section string, key string, value float64)
	SetInt(section string, key string, value int)

	//delete basic conf
	Del(section string, key string)

	//获取一个section的配置管理
	GetSectionManager(section string) SectionConfManager
}

type SectionConfManager interface {
	//read basic conf from tas.conf file
	//返回section组下的key的值, 若未配置, 则返回默认值defv
	GetString(key string, defaultValue string) string
	GetBool(key string, defaultValue bool) bool
	GetDouble(key string, defaultValue float64) float64
	GetInt(key string, defaultValue int) int

	//set basic conf to tas.conf file
	SetString(key string, value string)
	SetBool(key string, value bool)
	SetDouble(key string, value float64)
	SetInt(key string, value int)

	//delete basic conf
	Del(key string)
}

type ConfFileManager struct {
	path string
	dict ini.Dict
	lock sync.RWMutex
}

type SectionConfFileManager struct {
	section string
	cfm     ConfManager
}

var GlobalConf ConfManager

func InitConf(path string) {
	if GlobalConf == nil {
		GlobalConf = NewConfINIManager(path)
	}
}

func NewConfINIManager(path string) ConfManager {
	cs := &ConfFileManager{
		path: path,
	}

	_, err := os.Stat(path)

	if err != nil && os.IsNotExist(err) {
		_, err = os.Create(path)
		if err != nil {
			DefaultLogger.Errorf("Failed to init the config manager: ", err)
			// exit if init config manager failed with io error
			panic(err)
		}
	} else if err != nil {
		DefaultLogger.Errorf("Failed to init the config manager: ", err)
		// exit if init config manager failed with io error
		panic(err)
	}
	cs.dict = ini.MustLoad(path)

	return cs
}

func (cs *ConfFileManager) GetSectionManager(section string) SectionConfManager {
	return &SectionConfFileManager{
		section: section,
		cfm:     cs,
	}
}

func (sfm *SectionConfFileManager) GetString(key string, defaultValue string) string {
	return sfm.cfm.GetString(sfm.section, key, defaultValue)
}

func (sfm *SectionConfFileManager) GetBool(key string, defaultValue bool) bool {
	return sfm.cfm.GetBool(sfm.section, key, defaultValue)
}

func (sfm *SectionConfFileManager) GetDouble(key string, defaultValue float64) float64 {
	return sfm.cfm.GetDouble(sfm.section, key, defaultValue)
}

func (sfm *SectionConfFileManager) GetInt(key string, defaultValue int) int {
	return sfm.cfm.GetInt(sfm.section, key, defaultValue)
}

func (sfm *SectionConfFileManager) SetString(key string, value string) {
	sfm.cfm.SetString(sfm.section, key, value)
}

func (sfm *SectionConfFileManager) SetBool(key string, value bool) {
	sfm.cfm.SetBool(sfm.section, key, value)
}

func (sfm *SectionConfFileManager) SetDouble(key string, value float64) {
	sfm.cfm.SetDouble(sfm.section, key, value)
}

func (sfm *SectionConfFileManager) SetInt(key string, value int) {
	sfm.cfm.SetInt(sfm.section, key, value)
}

func (sfm *SectionConfFileManager) Del(key string) {
	sfm.cfm.Del(sfm.section, key)
}

func (cs *ConfFileManager) GetString(section string, key string, defaultValue string) string {
	cs.lock.RLock()
	defer cs.lock.RUnlock()

	if v, ok := cs.dict.GetString(strings.ToLower(section), strings.ToLower(key)); ok {
		return v
	}
	return defaultValue
}

func (cs *ConfFileManager) GetBool(section string, key string, defaultValue bool) bool {
	cs.lock.RLock()
	defer cs.lock.RUnlock()

	if v, ok := cs.dict.GetBool(strings.ToLower(section), strings.ToLower(key)); ok {
		return v
	}
	return defaultValue
}

func (cs *ConfFileManager) GetDouble(section string, key string, defaultValue float64) float64 {
	cs.lock.RLock()
	defer cs.lock.RUnlock()

	if v, ok := cs.dict.GetDouble(strings.ToLower(section), strings.ToLower(key)); ok {
		return v
	}
	return defaultValue
}

func (cs *ConfFileManager) GetInt(section string, key string, defaultValue int) int {
	cs.lock.RLock()
	defer cs.lock.RUnlock()

	if v, ok := cs.dict.GetInt(strings.ToLower(section), strings.ToLower(key)); ok {
		return v
	}
	return defaultValue
}

func (cs *ConfFileManager) SetString(section string, key string, value string) {
	cs.update(func() {
		cs.dict.SetString(strings.ToLower(section), strings.ToLower(key), value)
	})
}

func (cs *ConfFileManager) SetBool(section string, key string, value bool) {
	cs.update(func() {
		cs.dict.SetBool(strings.ToLower(section), strings.ToLower(key), value)
	})
}

func (cs *ConfFileManager) SetDouble(section string, key string, value float64) {
	cs.update(func() {
		cs.dict.SetDouble(strings.ToLower(section), strings.ToLower(key), value)
	})
}

func (cs *ConfFileManager) SetInt(section string, key string, value int) {
	cs.update(func() {
		cs.dict.SetInt(strings.ToLower(section), strings.ToLower(key), value)
	})
}

func (cs *ConfFileManager) Del(section string, key string) {
	cs.update(func() {
		cs.dict.Delete(strings.ToLower(section), strings.ToLower(key))
	})
}

func (cs *ConfFileManager) update(updator func()) {
	cs.lock.Lock()
	defer cs.lock.Unlock()

	updator()
	cs.store()
}

func (cs *ConfFileManager) store() {
	err := ini.Write(cs.path, &cs.dict)
	if err != nil {

	}
}
