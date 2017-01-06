package main

import (
	"os"
	"fmt"
	"github.com/flosch/pongo2"
	"io/ioutil"
	"regexp"
	"strings"
	"os/exec"
	"bytes"
)

var reg = regexp.MustCompile(`\btype\s*([A-Z]\S*?)\s*struct\s*\{`)
var pkgReg = regexp.MustCompile(`\bpackage\s*(\w+)`)
var nocacheComments = regexp.MustCompile(`//nocache: (.*)?\n`)

type GoString []string

func (gs GoString) String() (string) {
	buf := bytes.NewBuffer(nil)
	buf.WriteString("[]string{")
	for _, v := range gs {
		buf.WriteString("\"")
		buf.WriteString(v)
		buf.WriteString("\"")
	}
	buf.WriteString("}")
	return buf.String()
}

type CacheObject struct {
	Name           string
	NocacheMethods GoString
}

func main() {
	gopath := os.Getenv("GOPATH")
	gopathes := strings.Split(gopath, string(os.PathListSeparator))

	workdir, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	for _, v := range gopathes {
		if strings.HasPrefix(workdir, v) {
			gopath = v
			break
		}
	}
	if strings.HasSuffix(gopath, string(os.PathSeparator)) {
		gopath = gopath[:len(gopath) - 1]
	}
	if strings.HasSuffix(workdir, string(os.PathSeparator)) {
		workdir = workdir[:len(workdir) - 1]
	}
	pkgPath := workdir[len(gopath) + 5:strings.LastIndex(workdir, string(os.PathSeparator))]

	tempDir := workdir + "/cached_gen_temp"
	os.Mkdir(tempDir, 0777)

	dir, err := os.Open(workdir)
	if err != nil {
		panic(err)
	}
	files, err := dir.Readdir(-1)
	dir.Close()
	if err != nil {
		panic(err)
	}
	pkgName := ""
	list := map[string]CacheObject{}
	for _, f := range files {
		if !f.IsDir() && strings.HasSuffix(f.Name(), ".go") && !strings.HasSuffix(f.Name(), "_test.go") {
			data, err := ioutil.ReadFile("./" + f.Name())
			if err != nil {
				continue
			}
			m := pkgReg.FindSubmatch(data)
			if len(m) > 1 {
				fmt.Println(string(m[1]))
				pkgName = string(m[1])
			}
			m = nocacheComments.FindSubmatch(data)
			nocacheMethods := []string{}
			if len(m) > 1 {
				fmt.Println(string(m[1]))
				nocacheMethods = strings.Split(string(m[1]), " ")
			}
			matches := reg.FindAllSubmatch(data, -1)
			for _, v := range matches {
				fmt.Println(string(v[1]))
				if co, ok := list[string(v[1])]; ok {
					co.NocacheMethods = append(co.NocacheMethods, nocacheMethods...)
				} else {
					list [string(v[1])] = CacheObject{
						Name:          string(v[1]),
						NocacheMethods:GoString(nocacheMethods),
					}
				}

			}

		}
	}
	pkgPath = pkgPath + "/" + pkgName
	proxypkg := pkgName

	tpl := `
package main
import "github.com/cocotyty/cachegen"
import "{{fullPkg}}"
import "io/ioutil"
func main(){
{% for name,li in list %}
	run(&{{proxypkg}}.{{li.Name}}{},"{{pkgname}}","{{fullpath}}/proxy_cached_{{li.Name}}.go",{{li.NocacheMethods|stringformat:"%s"|safe}})
{% endfor %}
}
func run(v interface{},pkgname string,fullpath string,nocacheMethod []string){
	data,ok:=cachegen.GenCache(v,pkgname,nocacheMethod...)
	if ok{
		ioutil.WriteFile(fullpath,data,0777)
	}
}
	`
	tp, err := pongo2.FromString(tpl)
	if err != nil {
		panic(err)
	}
	cachedGenDir := workdir
	//err = os.Mkdir(cachedGenDir, 0777)
	//if err != nil {
	//	f, err := os.Open(cachedGenDir)
	//	if err != nil {
	//		panic(err)
	//	}
	//	files, err := f.Readdir(-1)
	//	if err != nil {
	//		panic(err)
	//	}
	//	f.Close()
	//	for _, v := range files {
	//		if !v.IsDir()  && strings.HasPrefix(v.Name(),"proxy_cached_") &&strings.HasSuffix(v.Name(),".go"){
	//			os.Remove(cachedGenDir + "/" + v.Name())
	//		}
	//	}
	//}

	f, err := os.Create(tempDir + "/gen.go")
	if err != nil {
		panic(err)
	}

	tp.ExecuteWriter(pongo2.Context{
		"pkgname":   pkgName,
		"list":      list,
		"fullpath":  cachedGenDir,
		"fullPkg":   pkgPath,
		"proxypkg":  proxypkg,
	}, f)

	f.Close()
	cmd := exec.Command("go", "run", tempDir+"/gen.go")
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	err = cmd.Run()
	if err != nil {
		panic(err)
	}
	os.RemoveAll(tempDir)
	cmd = exec.Command("go", "fmt")
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Dir = cachedGenDir
	cmd.Run()
}
