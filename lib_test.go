package gluajson

import (
	"testing"

	"github.com/yuin/gopher-lua"
)

const code = `local function encode(...) local j = json.encode(...) print(j) return j end
local function mark(value) return json.decode(encode(value), true) end
assert(encode(nil) == "null")
assert(encode{} == "{}")
assert(encode{0} == "[0]")
assert(encode"qq" == "\"qq\"")
assert(encode(json.array{}) == "[]")
assert(encode(json.raw"qq") == "qq")
local t = json.decode(encode(json.object{[1] = 2, aaa = "b\"bb"}))
print(t)
assert(t["1"] == 2)
assert(t["aaa"] == "b\"bb")

local null, marker = json.unmark(mark(null))
assert(marker == json.raw)
assert(null == "null")

local _, marker = json.unmark(mark({}))
assert(marker == json.object)

local empty, marker = json.unmark(mark(json.array{}))
assert(marker == json.array)
assert(#empty == 0)

local num, marker = json.unmark(mark(91.5))
assert(marker == json.raw)
assert(num == "91.5")

assert(encode(data) == "{\"a\":\"242\"}")
assert(encode({{{1,2,{a="b"},4}}}) == "[[[1,2,{\"a\":\"b\"},4]]]")
`

func Test(t *testing.T) {
	ls := lua.NewState(lua.Options{})
	Loader(ls)
	ls.Env.RawSetString("json", ls.Get(1))
	ls.SetTop(0)

	ls.Env.RawSetString("data", &lua.LUserData{
		Value:     map[string]string{"a": "242"},
		Env:       ls.Env,
		Metatable: nil,
	})

	if err := ls.DoString(code); err != nil {
		t.Log(err)
		t.FailNow()
	}
}