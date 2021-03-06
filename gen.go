package cachegen

import (
	"reflect"
	"os"
	"strings"
	"fmt"
	"bytes"
	"strconv"
)

func writePkg(set map[string]bool, fuckType reflect.Type) {

	switch fuckType.Kind() {
	case reflect.Array, reflect.Slice, reflect.Ptr, reflect.Chan, reflect.Map:
		writePkg(set, fuckType.Elem())
		return
	}

	set[fuckType.PkgPath()] = true
}
func GenCache(v interface{}, pkgname string, nocacheMethods ...string) ([]byte, bool) {
	nocacheSet := map[string]bool{}
	for _, v := range nocacheMethods {
		nocacheSet[v] = true
	}
	gopath := os.Getenv("GOPATH")
	gopathes := strings.Split(gopath, string(os.PathListSeparator))
	gopath = gopathes[0]
	typ := reflect.TypeOf(v)
	l := typ.NumMethod()
	src := bytes.NewBuffer(nil)
	head := bytes.NewBuffer(nil)
	fmt.Println(typ.Elem().String())
	fullpkg := typ.Elem().PkgPath()
	temp := strings.Split(typ.Elem().String(), ".")
	pkgSet := map[string]bool{"github.com/cocotyty/summer":true, "github.com/cocotyty/cachegen/itfc":true, typ.Elem().PkgPath():true}
	pkg := pkgname
	realTypName := temp[1]
	typname := "proxyCached" + temp[1]
	head.WriteString(`package `)
	head.WriteString(pkg)
	head.WriteRune('\n')
	src.WriteString("\n func init(){\n summer.Put(&" + typname + "{}) \n}\ntype " + typname + " struct{\nCache itfc.Cache `sm:\"*\"`\nParent *" + realTypName + " `sm:\"*\"`\n}\n")
	var errForType error = nil
	fmt.Println(reflect.TypeOf(&errForType).Elem())
	methodNum := 0
	for i := 0; i < l; i++ {
		method := typ.Method(i)
		if nocacheSet[method.Name] {
			continue
		}
		if !strings.HasSuffix(method.Name, "WithCache") && method.Type.NumOut() == 2 && method.Type.Out(1) == reflect.TypeOf(&errForType).Elem() {
			methodNum++
			src.WriteString("\nfunc (_self *")
			src.WriteString(typname)
			src.WriteString(") ")
			src.WriteString(method.Name)
			src.WriteString(`(`)
			inNum := method.Type.NumIn()
			for j := 1; j < inNum; j++ {
				inArg := method.Type.In(j)
				src.WriteString("v" + strconv.Itoa(j) + " ")
				inStr:=""
			FLAG:
				if inArg.Kind() == reflect.Ptr {
					src.WriteString("*")
					inArg = inArg.Elem()
					goto FLAG
				}
				if inArg.PkgPath() == fullpkg {
					n := inArg.String()
					inStr += n[strings.Index(n, ".")+1:]
				} else {
					inStr += inArg.String()
				}
				writePkg(pkgSet, inArg)
				if method.Type.IsVariadic() && j == inNum-1 {
					src.WriteString("..." + inArg.Elem().String())
				} else {
					src.WriteString(inStr)
				}
				if j != inNum-1 {
					src.WriteByte(',')
				}
			}

			src.WriteString(")(r0 ")
			outResult := method.Type.Out(0)
			outStr := ""
		FLAG2:
			if outResult.Kind() == reflect.Ptr {
				outStr += "*"
				outResult = outResult.Elem()
				goto FLAG2
			}
			if outResult.Kind() == reflect.Slice {
				outStr += "[]"
				outResult = outResult.Elem()
				goto FLAG2
			}
			writePkg(pkgSet, outResult)
			if outResult.PkgPath() == fullpkg {
				n := outResult.String()
				outStr += n[strings.Index(n, ".")+1:]
			} else {
				outStr += outResult.String()
			}
			src.WriteString(outStr)
			src.WriteString(",")
			src.WriteString("err error")
			src.WriteString("){\n")
			src.WriteString("   ")
			src.WriteString(`v,err:=_self.Cache.Find(func() (interface{}, error) {
			return _self.Parent.`)
			src.WriteString(method.Name)
			src.WriteString("(")
			for j := 1; j < inNum; j++ {
				if method.Type.IsVariadic() && j == inNum-1 {
					src.WriteString("v" + strconv.Itoa(j) + " ...")
				} else {
					src.WriteString("v" + strconv.Itoa(j))
				}
				if j < inNum-1 {
					src.WriteString(",")
				}
			}
			src.WriteString(")")
			src.WriteString("\n   },")
			src.WriteString("\"")
			src.WriteString(typname)
			src.WriteString(".")
			src.WriteString(method.Name)
			src.WriteString("\",")
			for j := 1; j < inNum; j++ {

				src.WriteString("v" + strconv.Itoa(j))
				if j < inNum-1 {
					src.WriteString(",")
				}
			}
			src.WriteString(")\n")
			src.WriteString("   if err!=nil{  return }\n   r0=v.(")
			src.WriteString(outStr)
			src.WriteString(")\n   return\n}")
		}
	}

	for pkgPath := range pkgSet {
		if pkgPath == "" || pkgPath == fullpkg {
			continue
		}
		head.WriteString("import \"")
		head.WriteString(pkgPath)
		head.WriteString("\"\n")
	}
	fmt.Println("method num:", methodNum, temp)
	if methodNum == 0 {
		return nil, false
	}
	head.Write(src.Bytes())
	return head.Bytes(), true
}
