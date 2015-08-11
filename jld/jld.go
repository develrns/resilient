/*
Package jld provides a few JSON LD utility functions.
*/
package jld

import (
	"fmt"
	"net/url"

	"github.com/develrns/resilient/log"

	"github.com/kazarena/json-gold/ld"
)

var (
	logger   = log.Logger()
	emptyCtx = ld.NewContext(nil, ld.NewJsonLdOptions(""))

	//IDP is the @id PropID
	IDP = NewPropID("@id", "")

	//TypeP is the @type PropID
	TypeP = NewPropID("@type", "")

	//ValueP is the @value PropID
	ValueP = NewPropID("@value", "")

	//CtxP is the @context PropID
	CtxP = NewPropID("@context", "")
)

type (
	//TypeBase is an identifier base for a domain of JSON LD type identifiers.
	//Typically it ends in a "#" making it a domain containing fragment IDs.
	TypeBase string

	//TypeID holds the relative and URI forms of a JSON LD type identifier.
	TypeID struct {
		uri string
	}

	//PropBase is an identifier base for a domain of JSON LD property identifiers.
	//Typically it ends in a "#" making it a domain containing fragment IDs.
	PropBase string

	//PropID holds the relative and URI forms of a JSON LD property identifier.
	PropID struct {
		uri string
	}

	//IDer is an interface for accessing a TypeID's or PropID's relative and URI values.
	IDer interface {
		URI()
	}
)

var (
	bngen = ld.NewBlankNodeIDGenerator()
)

//NewTypeBase creates a new TypeBase.
func NewTypeBase(base string) TypeBase {
	var tb = TypeBase(base)
	return tb
}

//Str returns a TypeBase string.
func (tb TypeBase) Str() string {
	return string(tb)
}

//NewTypeID creates a new TypeID. If TypeBase is nil, the id is the ID's URI.
func NewTypeID(id string, base TypeBase) TypeID {
	var (
		uri string
		err error
	)

	if base == "" {
		uri = id
	} else {
		uri = string(base) + id
	}
	_, err = url.Parse(uri)
	if err != nil {
		panic("Bad TypeID")
	}
	return TypeID{uri: uri}
}

//URI returns a TypeID's URI.
func (tid TypeID) URI() string {
	return tid.uri
}

//NewPropBase creates a new PropBase.
func NewPropBase(base string) PropBase {
	var pb = PropBase(base)
	return pb
}

//Str returns a PropBase string.
func (pb PropBase) Str() string {
	return string(pb)
}

//NewPropID creates a new PropID. If PropBase is nil, the id is the ID's URI.
func NewPropID(id string, base PropBase) PropID {
	var (
		uri string
		err error
	)
	if base == "" {
		uri = id
	} else {
		uri = string(base) + id
	}
	_, err = url.Parse(uri)
	if err != nil {
		panic("Bad PropID")
	}
	return PropID{uri: uri}
}

//URI returns a PropID's URI.
func (pid PropID) URI() string {
	return pid.uri
}

/*
BlankID creates a blank node identifier unique within this process.
*/
func BlankID() string {
	return bngen.GenerateBlankNodeIdentifier("")
}

/*
NewV creates a typed value object. The value may be a bool,
int, float32, float64 or string value. Any other type of value returns a value object with @value nil.
*/
func NewV(t TypeID, v interface{}) map[string]interface{} {
	valobj := make(map[string]interface{}, 2)
	valobj["@type"] = t.uri
	switch v.(type) {
	case bool, int, float32, float64, string:
		valobj["@value"] = v
	default:
		valobj["@value"] = nil
	}
	return valobj
}

/*
NewN creates a node with @id and @type properties. If id is blank a blank node of the type is created.
*/
func NewN(id string, t TypeID) map[string]interface{} {
	var (
		node = make(map[string]interface{}, 2)
		err  error
	)

	node["@type"] = t.uri

	switch id {
	case "":
		node["@id"] = bngen.GenerateBlankNodeIdentifier("")
	default:
		_, err = url.Parse(id)
		if err != nil {
			panic("Bad ID")
		}
		node["@id"] = id
	}
	return node
}

/*
AddN adds an id and type to an existing map. This simplifies creating a node from a composite literal.
*/
func AddN(input interface{}, id string, t TypeID) {
	var (
		node         map[string]interface{}
		okID, okType bool
		err          error
	)

	switch input.(type) {
	case map[string]interface{}:
		node = input.(map[string]interface{})
		_, okID = node["@id"]
		_, okType = node["@type"]
		if okID || okType {
			panic("AddN to existing node")
		}
		node["@type"] = t.uri

		switch id {
		case "":
			node["@id"] = bngen.GenerateBlankNodeIdentifier("")
		default:
			_, err = url.Parse(id)
			if err != nil {
				panic("Bad ID")
			}
			node["@id"] = id
		}
	}
}

/*
NewL creates a list object containing the slice or interface.
*/
func NewL(s interface{}) map[string]interface{} {
	listobj := make(map[string]interface{}, 1)
	listobj["@list"] = s
	return listobj
}

/*
GetP gets the property of a node
*/
func GetP(input interface{}, propID PropID) (interface{}, bool) {
	var (
		node  map[string]interface{}
		propI interface{}
		ok    bool
	)

	node, ok = input.(map[string]interface{})
	if !ok {
		return nil, false
	}
	propI, ok = node[propID.uri]
	if !ok {
		return nil, false
	}
	return propI, true
}

/*
GetN gets the property of a node if it is a node
*/
func GetN(input interface{}, propID PropID) (map[string]interface{}, bool) {
	var (
		node  map[string]interface{}
		propI interface{}
		ok    bool
	)

	node, ok = input.(map[string]interface{})
	if !ok {
		return nil, false
	}
	propI, ok = node[propID.uri]
	if !ok {
		return nil, false
	}
	if !ld.IsNode(propI) {
		return nil, false
	}
	return propI.(map[string]interface{}), true
}

/*
GetNtype gets the property of a node if it is a node of the requested type
*/
func GetNtype(input interface{}, propID PropID, typeID TypeID) (map[string]interface{}, bool) {
	var (
		node  map[string]interface{}
		propI interface{}
		ok    bool
	)

	node, ok = input.(map[string]interface{})
	if !ok {
		return nil, false
	}
	propI, ok = node[propID.uri]
	if !ok {
		return nil, false
	}
	if !IsNtype(propI, typeID) {
		return nil, false
	}
	return propI.(map[string]interface{}), true
}

/*
GetNRef returns the id of a node reference if the input is one.
*/
func GetNRef(input interface{}) (string, bool) {
	var (
		nref map[string]interface{}
	)

	if !ld.IsNodeReference(input) {
		return "", false
	}
	nref = input.(map[string]interface{})
	return nref["@id"].(string), true
}

/*
GetSet gets the property of a node if it is a set or singleton as a slice of interface{}
*/
func GetSet(input interface{}, propID PropID) ([]interface{}, bool) {
	var (
		node  map[string]interface{}
		propI interface{}
		array []interface{}
		slice []interface{}
		ok    bool
	)

	node, ok = input.(map[string]interface{})
	if !ok {
		return nil, false
	}
	propI, ok = node[propID.uri]
	if !ok {
		return nil, false
	}
	switch propI.(type) {
	case []interface{}:
		return propI.([]interface{}), true
	case nil:
		return nil, true
	default:
		//If the value is a singleton, convert it to a singleton slice
		array = make([]interface{}, 1)
		slice = array[:]
		slice[0] = propI
		node[propID.uri] = slice
		return slice, true
	}
}

/*
GetList gets the slice value of a node's list property if it is a list. If the value of the list is an array, it is returned.
If not, the value is wrapped in an array and returned. The value of the list is reset
*/
func GetList(input interface{}, propID PropID) ([]interface{}, bool) {
	var (
		node    map[string]interface{}
		listI   interface{}
		listObj map[string]interface{}
		listVI  interface{}
		array   []interface{}
		slice   []interface{}
		ok      bool
	)

	node, ok = input.(map[string]interface{})
	if !ok {
		return nil, false
	}
	listI, ok = node[propID.uri]
	if !ok {
		return nil, false
	}
	listObj, ok = listI.(map[string]interface{})
	if !ok {
		return nil, false
	}
	listVI, ok = listObj["@list"]
	if !ok {
		return nil, false
	}
	switch listVI.(type) {
	case []interface{}:
		return listVI.([]interface{}), true
	case nil:
		return nil, true
	default:
		//If the value is a singleton, convert it to a singleton slice
		array = make([]interface{}, 1)
		slice = array[:]
		slice[0] = listVI
		listObj["@list"] = slice
		return slice, true
	}

}

/*
GetVtype gets the value of a node's typed value object if it is a value object of the requested type.
*/
func GetVtype(input interface{}, propID PropID, typeID TypeID) (interface{}, bool) {
	var (
		node  map[string]interface{}
		propI interface{}
		val   interface{}
		ok    bool
	)

	node, ok = input.(map[string]interface{})
	if !ok {
		return nil, false
	}
	propI, ok = node[propID.uri]
	if !ok {
		return nil, false
	}
	if !IsVtype(propI, typeID) {
		return nil, false
	}
	val = propI.(map[string]interface{})["@value"]
	return val, true
}

/*
GetString gets the property of a node if it is a string
*/
func GetString(input interface{}, propID PropID) (string, bool) {
	var (
		node  map[string]interface{}
		propI interface{}
		propN map[string]interface{}
		ok    bool
	)

	node, ok = input.(map[string]interface{})
	if !ok {
		return "", false
	}
	propI, ok = node[propID.uri]
	if !ok {
		return "", false
	}
	switch propI.(type) {
	case map[string]interface{}:
		propN = propI.(map[string]interface{})
		propI, ok = propN["@value"]
		if !ok {
			return "", false
		}
		switch propI.(type) {
		case string:
			return propI.(string), true
		default:
			return "", false
		}
	case string:
		return propI.(string), true
	default:
		return "", false
	}
}

/*
GetBool gets the property of a node if it is a boolean
*/
func GetBool(input interface{}, propID PropID) (bool, bool) {
	var (
		node  map[string]interface{}
		propI interface{}
		propN map[string]interface{}
		ok    bool
	)

	node, ok = input.(map[string]interface{})
	if !ok {
		return false, false
	}
	propI, ok = node[propID.uri]
	if !ok {
		return false, false
	}
	switch propI.(type) {
	case map[string]interface{}:
		propN = propI.(map[string]interface{})
		propI, ok = propN["@value"]
		if !ok {
			return false, false
		}
		switch propI.(type) {
		case bool:
			return propI.(bool), true
		default:
			return false, false
		}
	case bool:
		return propI.(bool), true
	default:
		return false, false
	}
}

/*
IsNref returns true if the input is a JSON LD node reference.
*/
func IsNref(input interface{}) bool {
	return ld.IsNodeReference(input)
}

/*
IsNtype is true if the input is a node and it is of type t.
*/
func IsNtype(input interface{}, t TypeID) bool {
	var (
		n  map[string]interface{}
		tv interface{}
		ok bool
	)

	if !ld.IsNode(input) {
		return false
	}
	n = input.(map[string]interface{})

	tv, ok = n["@type"]
	if !ok {
		return false
	}

	switch tv.(type) {
	case string:
		return t.uri == tv.(string)
	case []string:
		for _, typeval := range tv.([]string) {
			if t.uri == typeval {
				return true
			}
		}
	}
	return false
}

/*
IsVtype is true if the input is a typed value object and it is of type t.
*/
func IsVtype(input interface{}, t TypeID) bool {
	var (
		n  map[string]interface{}
		tv interface{}
		ok bool
	)

	switch input.(type) {
	case map[string]interface{}:
		n = input.(map[string]interface{})
	default:
		return false
	}

	tv, ok = n["@type"]
	if !ok {
		return false
	}

	switch tv.(type) {
	case string:
		return t.uri == tv.(string)
	default:
		return false
	}
}

/*
IsVval is true if the input is an untyped value object or primitive with the value v.
*/
func IsVval(input interface{}, v interface{}) bool {
	var (
		valobj map[string]interface{}
		vv     interface{}
		ok     bool
	)

	switch input.(type) {
	case map[string]interface{}:
		valobj = input.(map[string]interface{})
		_, ok = valobj["@type"]
		if ok {
			return false
		}
		vv, ok = valobj["@value"]
		if !ok {
			return false
		}
	default:
		vv = input
	}

	switch vv.(type) {
	case string:
		switch v.(type) {
		case string:
			return v.(string) == vv.(string)
		default:
			return false
		}
	case bool:
		switch v.(type) {
		case bool:
			return v.(bool) == vv.(bool)
		default:
			return false
		}
	case int:
		switch v.(type) {
		case int:
			return v.(int) == vv.(int)
		default:
			return false
		}
	case float64:
		switch v.(type) {
		case float64:
			return v.(float64) == vv.(float64)
		default:
			return false
		}
	case nil:
		switch v.(type) {
		case nil:
			return true
		default:
			return false
		}
	default:
		return false
	}
}

/*
IsVtypeval is true if the input is a typed value object with the type t and value v.
*/
func IsVtypeval(input interface{}, t TypeID, v interface{}) bool {
	var (
		valobj map[string]interface{}
		tv     interface{}
		vv     interface{}
		ok     bool
	)

	switch input.(type) {
	case map[string]interface{}:
		valobj = input.(map[string]interface{})
	default:
		return false
	}

	tv, ok = valobj["@type"]
	if !ok {
		return false
	}

	switch tv.(type) {
	case string:
		if t.uri != tv.(string) {
			return false
		}
	default:
		return false
	}

	vv, ok = valobj["@value"]
	if !ok {
		return false
	}

	return v == vv

}

/*
IsVequal is true the two typed or untyped value objects are equal
*/
func IsVequal(input1, input2 interface{}) bool {
	var (
		valobj1, valobj2   map[string]interface{}
		len1, len2         int
		tv1, tv2, vv1, vv2 interface{}
		ok                 bool
	)

	switch input1.(type) {
	case map[string]interface{}:
		valobj1 = input1.(map[string]interface{})
	default:
		return false
	}

	switch input2.(type) {
	case map[string]interface{}:
		valobj2 = input2.(map[string]interface{})
	default:
		return false
	}

	len1 = len(valobj1)
	len2 = len(valobj2)
	if len1 != len2 {
		return false
	}

	if len1 == 2 {
		tv1, ok = valobj1["@type"]
		if !ok {
			return false
		}

		tv2, ok = valobj2["@type"]
		if !ok {
			return false
		}

		if tv1 != tv2 {
			return false
		}
	}

	vv1, ok = valobj1["@value"]
	if !ok {
		return false
	}

	vv2, ok = valobj2["@value"]
	if !ok {
		return false
	}

	return vv1 == vv2
}

/*
IsList is true if the input is a list object.
*/
func IsList(input interface{}) bool {
	var (
		list map[string]interface{}
		ok   bool
	)

	switch input.(type) {
	case map[string]interface{}:
		list = input.(map[string]interface{})
	default:
		return false
	}

	_, ok = list["@list"]
	if !ok {
		return false
	}

	return true

}

/*
AddType Adds a type to a node.
*/
func AddType(input interface{}, t TypeID) error {
	var (
		node map[string]interface{}
		set  []interface{}
		ok   bool
	)

	node, ok = input.(map[string]interface{})
	if !ok {
		return fmt.Errorf("Bad Node")
	}
	set, ok = GetSet(node, TypeP)
	if !ok {
		return fmt.Errorf("Bad Node @type")
	}
	set[len(set)] = t.uri
	return nil
}

/*
Append appends a singelton or slice or array to a node's set or list property. It returns the resulting slice.
*/
func Append(input interface{}, propID PropID, items ...interface{}) ([]interface{}, error) {
	var (
		node          map[string]interface{}
		slice         []interface{}
		newSlice      []interface{}
		listObj       map[string]interface{}
		okSet, okList bool
		ok            bool
	)

	node, ok = input.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("Bad Node")
	}
	slice, okSet = GetSet(node, propID)
	if okSet {
		newSlice = append(slice, items...)
		node[propID.uri] = newSlice
		return newSlice, nil
	}
	slice, okList = GetList(node, propID)
	if okList {
		newSlice = append(slice, items...)
		listObj = node[propID.uri].(map[string]interface{})
		listObj["@list"] = newSlice
		return newSlice, nil
	}
	return nil, fmt.Errorf("Bad Node")
}

/*
ApplyN applys the function to the nodes of the input.
If it is a set, the function is applied to its elements.
If it is a list, the function is applied to its elements.
If it is a singleton, the function is applied to it.
If the input is anything else, it is ignored.
If the function returns an error, apply terminates and returns this error.
*/
func ApplyN(f func(map[string]interface{}) error, input interface{}) error {
	var (
		slice []interface{}
		intr  interface{}
		node  map[string]interface{}
		err   error
		ok    bool
	)

	ok = IsList(input)
	if ok {
		intr = input.(map[string]interface{})["@list"]
		slice, ok = intr.([]interface{})
		if !ok {
			return nil
		}
	} else {
		slice, _ = input.([]interface{})
	}

	if slice != nil {
		for _, intr = range slice {
			node, ok = intr.(map[string]interface{})
			if ok {
				err = f(node)
				if err != nil {
					return err
				}
			}
		}
		return nil
	}

	node, ok = input.(map[string]interface{})
	if ok {
		err = f(node)
		if err != nil {
			return err
		}
		return nil
	}
	return nil
}

/*
Canonicalize filters and transforms an unmarshalled JSON LD graph int o a consistent form for processing. This is done in three steps:

1. Expand removes all context if any has been included.

2. Frame extracts only the node(s) or value object(s) of the types specified. The outgoing edges of these nodes are
fully 'unrolled'. If the unrolled edges include a node with multiple incoming edges, a copy of the node is attached to each edge.

3. Compact is used to remove array 'wrappers' from singleton arrays.

The input must be unmarshalled JSON.
If only one node matches the typeFilter, it is returned; if no nodes are matched, the result is nil; otherwise an array of the matched nodes are returned.
*/
func Canonicalize(input interface{}, typeFilter []TypeID) (interface{}, error) {
	var (
		ldapi        = ld.NewJsonLdApi()
		err          error
		frame        []interface{}
		obj          map[string]interface{}
		types        = make([]interface{}, len(typeFilter))
		expanded     interface{}
		framed       interface{}
		framedArray  []interface{}
		compactInput interface{}
		compacted    interface{}
	)

	for i, typeID := range typeFilter {
		types[i] = typeID.uri
	}
	obj = map[string]interface{}{"@type": types}
	frame = []interface{}{obj}

	expanded, err = ldapi.Expand(emptyCtx, "", input)
	if err != nil {
		return nil, err
	}

	/*
		   ld package issues:

		   	* NewJsonLdApi does not accept a JsonLdOptions parameter as it documents. Instead it appears that JsonLdOptions is given to only
			subset of JsonLdApi functions. This implies that only these functions make use of it.
			For instance, only these use a Document Loader to resolve remote context/document references.
			"NewJsonLdApi creates a new instance of JsonLdApi and initialises it with the given JsonLdOptions structure."

			* Frame does not process lists correctly. It appears it loses their content after they have been flattened and then does
		   	not later embed the content. Instead it results in a hanging node reference to their content.

		   	* The output of the Node jsonld module wraps a graph object around Frame output - this package does not.

		   	* It also does not do the 'empty context' compact as specified by the framing spec.

		   	* The spec is very unclear about how to construct the input frame and exactly what features it provides.
		   	The ld package doesn't provide any additional description.
	*/
	framed, err = ldapi.Frame(expanded, frame, nil)
	if err != nil {
		return nil, err
	}
	switch framed.(type) {
	case map[string]interface{}:
		compactInput = framed
	case []interface{}:
		framedArray = framed.([]interface{})
		switch len(framedArray) {
		case 0:
			return nil, nil
		case 1:
			compactInput = framedArray[0]
		default:
			compactInput = framed
		}
	default:
		return nil, nil
	}
	compacted, err = ldapi.Compact(emptyCtx, "", compactInput, true)
	if err != nil {
		return nil, err
	}
	return compacted, nil
}

/*
PrintDocument is the same as ld.PrintDocument - it prints the internal JSON LD Document as formatted JSON LD.
It's here to eliminate the need to import the ld package.
*/
func PrintDocument(msg string, document interface{}) {
	ld.PrintDocument(msg, document)
}
