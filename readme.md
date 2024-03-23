JSON library for [gopher-lua](https://github.com/yuin/gopher-lua) in PeerDB

Unfortunately existing JSON libraries for gopher-lua suffer from the existing issue of how to encode an empty table,
while this library offers a mechanism to specify how a particular table should be encoded if empty

These libraries also fail to integrate with UserData, which PeerDB scripting relies on heavily

----

gluajson exports `encode` & `decode`

To guide `encode`, there are 3 wrapper functions: `array`, `object`, `raw`

`array` / `table` can be used to direct whether an empty table should be encoded as an array or a table

`json.raw(string)` will interpolate `string` without encoding,
this can be useful if you want to wrap an already encoded json string into another json object

Passing `true` as the 2nd parameter to `decode` will decode json using markers,
arrays will be marked as array,
objects marked as object,
numbers marked as raw strings of their digits to preserve precision,
& nulls marked as `raw("null")`

To handle marked objects, there is a function `unmark` which for marked objects returns `inner, wrapper`,
while for unmarked objects the original parameter is returned