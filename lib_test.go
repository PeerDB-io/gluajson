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
assert(encode(json.array()) == "null")
assert(encode(json.raw"qq") == "qq")
assert(encode(data) == "{\"a\":\"242\"}")
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

local nested = encode({{{1,2,{a="b"},4}}})
assert(nested == "[[[1,2,{\"a\":\"b\"},4]]]")
nested = json.decode(nested)[1][1]
assert(nested[1] == 1)
assert(nested[2] == 2)
assert(nested[3]["a"] == "b")
assert(nested[4] == 4)

local t1 = setmetatable({}, {__json = function() return 1 end})
assert(json.encode(t1) == "1")
local t2 = setmetatable({}, {__json = function() return t1 end})
assert(json.encode(t2) == "1")
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