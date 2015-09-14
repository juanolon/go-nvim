// +build ignore
package main

import (
	"fmt"
	"io"
	"log"
	"net"
	"net/rpc"
	"os"
	"strings"
	"text/template"
	"unicode"
	"unicode/utf8"

	"github.com/ugorji/go/codec"
)

type ErrorTypesDefinitions map[string]map[string]int64
type TypesDefinitions map[string]map[string]int64
type FunctionDefinitions map[string]FunctionDesc

type FunctionDesc struct {
	Parameters []string
	Return     string
}

type NvimDefinitions struct {
	Fd FunctionDefinitions
	Td TypesDefinitions
	Ed ErrorTypesDefinitions
}

// TEMPLATE FUNCTIONS
func upperFirst(s string) string {
	if s == "" {
		return ""
	}
	r, n := utf8.DecodeRuneInString(s)
	return string(unicode.ToUpper(r)) + s[n:]
}

func parseMethod(s string) (string, []string) {
	parts := strings.Split(s, "_")
	return parts[0], parts[1:]
}

func methodName(s string) string {
	_, methodNames := parseMethod(s)
	var res string
	for _, name := range methodNames {
		res += upperFirst(name)
	}
	return res
}

func structName(s string) string {
	method, _ := parseMethod(s)
	return upperFirst(method)
}

var types = map[string]string{
	"String":              "[]byte",
	"ArrayOf(String)":     "[]string",
	"ArrayOf(Integer, 2)": "[]int",
	"Integer":             "int",
	"Boolean":             "bool",
	"Object":              "interface{}",
	"Buffer":              "BufferIdentifier",
	"ArrayOf(Buffer)":     "[]BufferIdentifier",
	"Window":              "WindowIdentifier",
	"ArrayOf(Window)":     "[]WindowIdentifier",
	"Tabpage":             "TabpageIdentifier",
	"ArrayOf(Tabpage)":    "[]TabpageIdentifier",
	"void":                "nil",
	"Array":               "[]interface{}",          // vim_get_api_info
	"Dictionary":          "map[string]interface{}", // vim_get_color_map
}

var returnTypes = map[string]string{
	"Buffer":  "Buffer",
	"Window":  "Window",
	"Tabpage": "Tabpage",
}

// getType tries to parse t and return
// it corresponding go type else //TODO return err
func getType(s string) string {
	t, ok := types[s]
	if !ok {
		log.Fatalf("Could not find type for %v", s)
	}
	return t
}

func getReturnType(s string) string {
	t, ok := returnTypes[s]
	if !ok {
		log.Fatalf("Could not find struct for %s", s)
	}
	return t
}

func parseParameters(p []string, named bool) string {
	ret := ""
	if named {
		for k, v := range p {
			ret += fmt.Sprintf("v%d %s", k, getType(v))
			if k != len(p)-1 {
				ret += ", "
			}
		}
	} else {
		for k, _ := range p {
			ret += fmt.Sprintf("v%d", k)
			if k != len(p)-1 {
				ret += ", "
			}
		}
	}
	return ret
}

func parseType(r string) string {
	return getType(r)
}

func parseStruct(r string) string {
	// t := getType(r)
	return getReturnType(r)
}

func isStruct(s string) bool {
	_, ok := returnTypes[s]
	return ok
}

// END TEMPLATE FUNCTIONS

func main() {
	socket := "/var/folders/x6/h18jf2xj10bgb1_jlf_fy0rr0000gn/T/nvimaGKXbS/0"

	conn, err := net.Dial("unix", socket)
	if err != nil {
		log.Fatal("fail to connect to server: ", err)
		return
	}

	// no extensions needed for now
	var h codec.MsgpackHandle
	h.RawToString = true
	h.WriteExt = true

	rpcCodec := codec.MsgpackSpecRpc.ClientCodec(conn, &h)
	client := rpc.NewClientWithCodec(rpcCodec)

	bapi := struct {
		Id          uint64 // what it's the id for?
		Definitions map[interface{}]interface{}
	}{
		0,
		nil,
	}
	args := codec.MsgpackSpecRpcMultiArgs{}
	err = client.Call("vim_get_api_info", args, &bapi)
	if err != nil {
		log.Fatal("rpc send cmd: ", err)
	}
	var api NvimDefinitions

	for cat, mapv := range bapi.Definitions {
		switch cat {
		case "functions":
			api.Fd = FunctionDecode(mapv)
		case "types":
			api.Td = TypeDecode(mapv)
		case "error_types":
			api.Ed = ErrorTypeDecode(mapv)
		default:
			fmt.Printf("%s {{}}\n", cat)
		}
	}
	// fmt.Println("=========================")
	// fmt.Printf("api: {{\n")
	// spew.Dump(api)
	// fmt.Printf("}}\n")
	// fmt.Println("=========================")

	funcMap := template.FuncMap{
		"uppercase":       upperFirst,
		"methodName":      methodName,
		"structName":      structName,
		"parseParameters": parseParameters,
		"parseType":       parseType,
		"parseStruct":     parseStruct,
		"isStruct":        isStruct,
	}
	t := template.Must(template.New("main").Funcs(funcMap).ParseGlob("gen/tpl/*.gotexttmpl"))
	if err != nil {
		log.Fatalf("tpl error: %+v", err)
	}
	var out io.Writer
	// out = os.Stdout
	out, err = os.OpenFile("api.go", os.O_WRONLY|os.O_TRUNC|os.O_CREATE, 0644)
	if err != nil {
		log.Fatalf("Could not generate gen_api.go file\n%#+v\n", err)
	}
	err = t.ExecuteTemplate(out, "main.gotexttmpl", api)
	if err != nil {
		log.Fatalf("Could not execute template:\n%+#v\n", err)
	}
}

func FunctionDecode(val interface{}) FunctionDefinitions {
	defs := val.([]interface{})
	fnDef := FunctionDefinitions{}
	for _, v := range defs {
		attrs, ok := v.(map[interface{}]interface{})
		if !ok {
			log.Fatalf("function dec: failed to convert %v %T to string", attrs, attrs)
		}
		// spew.Dump(attrs["name"])
		b := attrs["name"].([]byte)
		name := string(b)

		b = attrs["return_type"].([]byte)
		sreturn := string(b)

		var parameters []string
		par, ok := attrs["parameters"].([]interface{})
		if !ok {
			log.Fatalf("parameters: failed to convert %v %T to string")
		}
		// fmt.Println("printing parameters")
		for _, v := range par {
			val, ok := v.([]interface{})
			if !ok {
				log.Fatalf("parameter value: failed to convert %v %T to string", val, val)
			}
			b := val[0].([]byte)
			parameters = append(parameters, string(b))
		}

		fnDef[name] = FunctionDesc{
			Parameters: parameters,
			Return:     sreturn,
		}
		// spew.Dump(fnDef)
	}
	return fnDef
}

func TypeDecode(val interface{}) TypesDefinitions {
	definitions, ok := val.(map[interface{}]interface{})
	if !ok {
		log.Fatalf("definitions: failed to convert %v %T to string")
	}

	typesDef := TypesDefinitions{}
	for cat, mapv := range definitions {
		name, ok := cat.(string)
		if !ok {
			log.Fatalf("for definitions: failed to convert %v %T to string")
		}
		typesDef[name] = make(map[string]int64)

		members, ok := mapv.(map[interface{}]interface{})
		if !ok {
			log.Fatalf("definitions: failed to convert %v %T to string", members, members)
		}
		for k, v := range members {
			key := string(k.(string))
			val := int64(v.(int64))
			typesDef[name][key] = val
		}
	}
	return typesDef
}

func ErrorTypeDecode(val interface{}) ErrorTypesDefinitions {
	definitions, ok := val.(map[interface{}]interface{})
	if !ok {
		log.Fatalf("definitions: failed to convert %v %T to string")
	}

	errTypes := ErrorTypesDefinitions{}
	for cat, mapv := range definitions {
		name, ok := cat.(string)
		if !ok {
			log.Fatalf("for definitions: failed to convert %v %T to string")
		}
		errTypes[name] = make(map[string]int64)

		members, ok := mapv.(map[interface{}]interface{})
		if !ok {
			log.Fatalf("definitions: failed to convert %v %T to string", members, members)
		}
		for k, v := range members {
			key := string(k.(string))
			val := int64(v.(int64))
			errTypes[name][key] = val
		}
	}
	return errTypes
}
