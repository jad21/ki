package ki

import (
	"encoding/json"
	"strconv"
	// "github.com/gorilla/mux"
)

func (s *Context) DecodeJSON(data any) error {
	return json.NewDecoder(s.Request.Body).Decode(&data)
}

// func (s *Context) Vars() map[string]string {
// 	return mux.Vars(s.Request)
// }

func (s *Context) FormValue(key string) string {
	return s.Request.FormValue(key)
}

func (s *Context) PostFormValue(key string) string {
	return s.Request.PostFormValue(key)
}

func (s *Context) Has(key string) bool {
	return s.Request.Form.Has(key) || s.Request.PostForm.Has(key)
}

func (s *Context) GetStr(key string, valDefault ...string) string {
	if len(valDefault) > 0 {
		if !s.Has(key) {
			return valDefault[0]
		}
	}
	return s.FormValue(key)
}

func (s *Context) GetInt(key string, valDefault ...int) int {
	if len(valDefault) > 0 {
		if !s.Has(key) {
			return valDefault[0]
		}
	}
	v, _ := strconv.Atoi(s.FormValue(key))
	return v
}
func (s *Context) GetInt64(key string, valDefault ...int64) int64 {
	if len(valDefault) > 0 {
		if !s.Has(key) {
			return valDefault[0]
		}
	}
	v, _ := strconv.ParseInt(s.FormValue(key), 10, 64)
	return v
}

func (s *Context) GetUInt64(key string, valDefault ...uint64) uint64 {
	if len(valDefault) > 0 {
		if !s.Has(key) {
			return valDefault[0]
		}
	}
	v, _ := strconv.ParseUint(s.FormValue(key), 10, 64)
	return v
}

func (s *Context) GetFloat64(key string, valDefault ...float64) float64 {
	if len(valDefault) > 0 {
		if !s.Has(key) {
			return valDefault[0]
		}
	}
	v, _ := strconv.ParseFloat(s.FormValue(key), 64)
	return v
}

func (s *Context) GetBool(key string, valDefault ...bool) bool {
	if len(valDefault) > 0 {
		if !s.Has(key) {
			return valDefault[0]
		}
	}
	v, _ := strconv.ParseBool(s.FormValue(key))
	return v
}

func (s *Context) GetStrPtr(key string) *string {
	if !s.Has(key) {
		return nil
	}
	v := s.FormValue(key)
	return &v
}

func (s *Context) GetIntPtr(key string) *int {
	if !s.Has(key) {
		return nil
	}
	v, _ := strconv.Atoi(s.FormValue(key))
	return &v
}

func (s *Context) GetInt64Ptr(key string) *int64 {
	if !s.Has(key) {
		return nil
	}
	v, _ := strconv.ParseInt(s.FormValue(key), 10, 64)
	return &v
}

func (s *Context) GetUInt64Ptr(key string) *uint64 {
	if !s.Has(key) {
		return nil
	}
	v, _ := strconv.ParseUint(s.FormValue(key), 10, 64)
	return &v
}

func (s *Context) GetFloat64Ptr(key string) *float64 {
	if !s.Has(key) {
		return nil
	}
	v, _ := strconv.ParseFloat(s.FormValue(key), 64)
	return &v
}

func (s *Context) GetBoolPtr(key string) *bool {
	if !s.Has(key) {
		return nil
	}
	v, _ := strconv.ParseBool(s.FormValue(key))
	return &v
}

func (s *Context) GetStrArray(key string) (results []string) {
	if !s.Has(key) {
		return
	}
	return s.Request.Form[key]
}

func (s *Context) GetIntArray(key string) (results []int) {
	if !s.Has(key) {
		return
	}
	for _, v := range s.Request.Form[key] {
		r, _ := strconv.Atoi(v)
		results = append(results, r)
	}
	return
}
func (s *Context) GetInt64Array(key string) (results []int64) {
	if !s.Has(key) {
		return
	}
	for _, v := range s.Request.Form[key] {
		r, _ := strconv.ParseInt(v, 10, 64)
		results = append(results, r)
	}
	return
}

func (s *Context) GetUInt64Array(key string) (results []uint64) {
	if !s.Has(key) {
		return
	}
	for _, v := range s.Request.Form[key] {
		r, _ := strconv.ParseUint(v, 10, 64)
		results = append(results, r)
	}
	return
}

func (s *Context) GetFloat64Array(key string) (results []float64) {
	if !s.Has(key) {
		return
	}
	for _, v := range s.Request.Form[key] {
		r, _ := strconv.ParseFloat(v, 64)
		results = append(results, r)
	}
	return
}

func (s *Context) GetBoolArray(key string) (results []bool) {
	if !s.Has(key) {
		return
	}
	for _, v := range s.Request.Form[key] {
		r, _ := strconv.ParseBool(v)
		results = append(results, r)
	}
	return
}
