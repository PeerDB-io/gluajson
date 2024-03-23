package gluajson

import (
	"encoding/json"
	"strings"

	"github.com/yuin/gopher-lua"
)

type Array *lua.LTable
type Object *lua.LTable
type Raw lua.LString

func newLjArray(ls *lua.LState) Array {
	return Array(ls.OptTable(1, nil))
}

func newLjObject(ls *lua.LState) Object {
	return Object(ls.OptTable(1, nil))
}

func newLjRaw(ls *lua.LState) Raw {
	return Raw(ls.CheckString(1))
}

func ljMarker[T any](f func(ls *lua.LState) T) func(ls *lua.LState) int {
	return func(ls *lua.LState) int {
		ls.Push(&lua.LUserData{
			Value:     f(ls),
			Env:       ls.Env,
			Metatable: nil,
		})
		return 1
	}
}

func ljUnmark(ls *lua.LState) int {
	val := ls.Get(1)
	ud, ok := val.(*lua.LUserData)
	if ok {
		switch v := ud.Value.(type) {
		case Object:
			ls.Push((*lua.LTable)(v))
		case Array:
			ls.Push((*lua.LTable)(v))
		case Raw:
			ls.Push(lua.LString(v))
		default:
			ls.Push(val)
		}
		return 1
	} else {
		ls.Push(val)
		return 1
	}
}

func LuaJsonEncode(ls *lua.LState) int {
	var sb strings.Builder
	dupe := make(map[lua.LValue]struct{})
	v := ls.Get(1)
	ljEncode(ls, v, &sb, dupe)
	ls.Push(lua.LString(sb.String()))
	return 1
}

func markDupe(ls *lua.LState, dupe map[lua.LValue]struct{}, v lua.LValue) {
	_, has := dupe[v]
	if has {
		ls.RaiseError("object contained cycle")
	}
	dupe[v] = struct{}{}
}

func ljEncode(
	ls *lua.LState,
	value lua.LValue,
	sb *strings.Builder,
	dupe map[lua.LValue]struct{},
) {
	switch v := value.(type) {
	case *lua.LNilType:
		sb.WriteString("null")
	case lua.LBool:
		if v {
			sb.WriteString("true")
		} else {
			sb.WriteString("false")
		}
	case lua.LNumber:
		sb.WriteString(v.String())
	case lua.LString:
		str, err := json.Marshal(string(v))
		if err != nil {
			ls.RaiseError("json.Marshal failed to serialize string")
		}
		sb.Write(str)
	case *lua.LTable:
		vlen := ls.ObjLen(v)
		if vlen == 0 {
			ljEncodeTable(ls, v, sb, dupe)
		} else {
			ljEncodeArray(ls, v, vlen, sb, dupe)
		}
	case *lua.LUserData:
		switch ud := v.Value.(type) {
		case Array:
			if ud == nil {
				sb.WriteString("null")
			} else {
				vlen := ls.ObjLen(v)
				ljEncodeArray(ls, (*lua.LTable)(ud), vlen, sb, dupe)
			}
		case Object:
			if ud == nil {
				sb.WriteString("null")
			} else {
				ljEncodeTable(ls, (*lua.LTable)(ud), sb, dupe)
			}
		case Raw:
			sb.WriteString(string(ud))
		default:
			bytes, err := json.Marshal(&ud)
			if err != nil {
				ls.RaiseError(err.Error())
				return
			}
			sb.Write(bytes)
		}
	case *lua.LFunction:
		ls.RaiseError("Cannot encode " + v.Type().String())
		return
	}
}

func ljEncodeArray(
	ls *lua.LState,
	v *lua.LTable,
	vlen int,
	sb *strings.Builder,
	dupe map[lua.LValue]struct{},
) {
	markDupe(ls, dupe, v)
	sb.WriteByte('[')
	for i := range vlen {
		ljEncode(ls, v.RawGetInt(i+1), sb, dupe)
		if i < vlen-1 {
			sb.WriteByte(',')
		}
	}
	sb.WriteByte(']')
}

func ljEncodeTable(
	ls *lua.LState,
	v *lua.LTable,
	sb *strings.Builder,
	dupe map[lua.LValue]struct{},
) {
	markDupe(ls, dupe, v)
	sb.WriteByte('{')
	var key lua.LValue
	key = lua.LNil
	for {
		newkey, value := v.Next(key)
		if newkey == lua.LNil {
			break
		}
		if key != lua.LNil {
			sb.WriteByte(',')
		}
		key = newkey

		keyb, err := json.Marshal(key.String())
		if err != nil {
			ls.RaiseError(err.Error())
			return
		}
		sb.Write(keyb)
		sb.WriteByte(':')
		ljEncode(ls, value, sb, dupe)
	}
	sb.WriteByte('}')
}

type DecodeTarget struct {
	table *lua.LTable
	key   string
	index int
}

func (dt *DecodeTarget) Ingest(value lua.LValue) {
	switch {
	case dt.index > 0:
		dt.table.RawSetInt(dt.index, value)
		dt.index += 1
	case dt.index == 0:
		dt.key = value.String()
		dt.index = -1
	case dt.index == -1:
		dt.table.RawSetString(dt.key, value)
		dt.index = 0
	}
}

func LuaJsonDecode(ls *lua.LState) int {
	decoder := json.NewDecoder(strings.NewReader(ls.CheckString(1)))
	exact := ls.OptBool(2, false)
	if exact {
		decoder.UseNumber()
	}

	var stack []DecodeTarget
	for {
		tok, err := decoder.Token()
		if err != nil {
			ls.RaiseError(err.Error())
		}
		var val lua.LValue
		switch tk := tok.(type) {
		case json.Delim:
			if tk == '[' {
				stack = append(stack, DecodeTarget{
					table: ls.NewTable(),
					key:   "",
					index: 1,
				})
				continue
			}
			if tk == '{' {
				stack = append(stack, DecodeTarget{
					table: ls.NewTable(),
					key:   "",
					index: 0,
				})
				continue
			}
			last := stack[len(stack)-1]
			if exact {
				if last.index > 0 {
					val = &lua.LUserData{
						Value:     Array(last.table),
						Env:       ls.Env,
						Metatable: nil,
					}
				} else {
					val = &lua.LUserData{
						Value:     Object(last.table),
						Env:       ls.Env,
						Metatable: nil,
					}
				}
			} else {
				val = last.table
			}
			stack = stack[:len(stack)-1]
		case json.Number:
			val = &lua.LUserData{
				Value:     Raw(tk),
				Env:       ls.Env,
				Metatable: nil,
			}
		case float64:
			val = lua.LNumber(tk)
		case string:
			val = lua.LString(tk)
		case bool:
			val = lua.LBool(tk)
		case nil:
			if exact {
				val = &lua.LUserData{
					Value:     Raw("null"),
					Env:       ls.Env,
					Metatable: nil,
				}
			} else {
				val = lua.LNil
			}
		}
		if len(stack) == 0 {
			ls.Push(val)
			return 1
		} else {
			stack[len(stack)-1].Ingest(val)
		}
	}
}

func Loader(ls *lua.LState) int {
	object := ls.NewFunction(ljMarker(newLjObject))
	array := ls.NewFunction(ljMarker(newLjArray))
	raw := ls.NewFunction(ljMarker(newLjRaw))
	m := ls.NewTable()
	m.RawSetString("encode", ls.NewFunction(LuaJsonEncode))
	m.RawSetString("decode", ls.NewFunction(LuaJsonDecode))
	m.RawSetString("object", object)
	m.RawSetString("array", array)
	m.RawSetString("raw", raw)
	m.RawSetString("unmark", ls.NewFunction(func(ls *lua.LState) int {
		val := ls.Get(1)
		if ud, ok := val.(*lua.LUserData); ok {
			switch v := ud.Value.(type) {
			case Object:
				ls.Push((*lua.LTable)(v))
				ls.Push(object)
				return 2
			case Array:
				ls.Push((*lua.LTable)(v))
				ls.Push(array)
				return 2
			case Raw:
				ls.Push(lua.LString(v))
				ls.Push(raw)
				return 2
			}
		}
		ls.Push(val)
		return 1
	}))

	ls.Push(m)
	return 1
}