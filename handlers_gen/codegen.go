package main

import (
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"log"
	"os"
	"reflect"
	"strings"
)

func main() {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, os.Args[1], nil, parser.ParseComments)
	if err != nil {
		log.Fatal(err)
	}

	out, _ := os.Create(os.Args[2])

	genPackageAndImports(out, node.Name.Name)

	for _, f := range node.Decls {
		switch f := f.(type) {
		case *ast.GenDecl:
			processGenDecl(out, f)
		case *ast.FuncDecl:
			processFuncDecl(out, f)
		default:
			fmt.Printf("SKIP %T\n", f)
		}
	}

	genServeHTTP(out)
}

func getTypeName(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		if ident, ok := t.X.(*ast.Ident); ok {
			return "*" + ident.Name
		}
	case *ast.ArrayType:
		if ident, ok := t.Elt.(*ast.Ident); ok {
			return "[]" + ident.Name
		}
	}
	return ""
}

func genPackageAndImports(out io.Writer, packageName string) {
	fmt.Fprintf(out, `// generated with codegen
package %s

import "errors"
import "fmt"
import "net/http"
import "net/url"
import "slices"
import "strconv"
import "strings"

`, packageName)
}

func processGenDecl(out io.Writer, g *ast.GenDecl) {
	for _, spec := range g.Specs {
		currType, ok := spec.(*ast.TypeSpec)
		if !ok {
			fmt.Printf("SKIP %T is not ast.TypeSpec\n", spec)
			continue
		}

		currStruct, ok := currType.Type.(*ast.StructType)
		if !ok {
			fmt.Printf("SKIP %T is not ast.StructType\n", currStruct)
			continue
		}

		validationFields := 0
		for _, field := range currStruct.Fields.List {

			if field.Tag != nil {
				tag := reflect.StructTag(field.Tag.Value[1 : len(field.Tag.Value)-1])
				if tag.Get("apivalidator") != "" {
					validationFields++
				}
			}

		}

		if validationFields > 0 && validationFields == currStruct.Fields.NumFields() {
			genValidation(out, currStruct, currType.Name.Name)
		} else {
			fmt.Print("SKIP not enough 'apivalidator' tags\n", currStruct)
			continue
		}
	}
}

func genValidation(out io.Writer, currStruct *ast.StructType, structName string) {
	fmt.Fprintf(out, "func (p *%s) fillAndValidate(v url.Values) (err error) {\n", structName)
	fmt.Fprintln(out, "\t var param string")

	for _, field := range currStruct.Fields.List {
		fieldName := field.Names[0].Name
		fieldTypeName := getTypeName(field.Type)

		tagsStr := reflect.StructTag(field.Tag.Value[1 : len(field.Tag.Value)-1]).Get("apivalidator")
		tags := strings.Split(tagsStr, ",")

		validators := make(map[string]string)
		for _, tag := range tags {
			k, v, _ := strings.Cut(tag, "=")
			validators[k] = v
		}

		var paramName string
		if value, found := validators["paramname"]; found {
			paramName = value
		} else {
			paramName = strings.ToLower(fieldName)
		}
		fmt.Fprintf(out, "\n\tparam = v.Get(%q)\n", paramName)

		if _, found := validators["required"]; found {
			fmt.Fprintf(out, `
	if param == "" {
		return fmt.Errorf("%s must me not empty")
	}
`, paramName)
		}

		if defaultValue, found := validators["default"]; found {
			fmt.Fprintf(out, `
	if param == "" {
		p.%s = %q
	} else {
		p.%s = param
	}
`, fieldName, defaultValue, fieldName)
		} else {
			if fieldTypeName == "int" {
				fmt.Fprintf(out, `	intParam, err := strconv.Atoi(param)
	if err != nil {
		return fmt.Errorf("%s must be int")
	}
	p.%s = intParam
`, paramName, fieldName)
			} else {
				fmt.Fprintf(out, "\tp.%s = param\n", fieldName)
			}
		}

		if enumsStr, found := validators["enum"]; found {
			fmt.Fprintf(out, `
	enums := strings.Split(%q, "|")
	if !slices.Contains(enums, p.%s) {
		return fmt.Errorf("%s must be one of [%%s]", strings.Join(enums, ", "))
	}
`, enumsStr, fieldName, paramName)
		}

		if strValue, found := validators["min"]; found {
			tempStr := fmt.Sprintf("p.%s", fieldName)
			tempStr2 := ""
			if fieldTypeName == "string" {
				tempStr = fmt.Sprintf("len(%s)", tempStr)
				tempStr2 = "len "
			}
			fmt.Fprintf(out, `
	if %s < %s {
		return fmt.Errorf("%s %smust be >= %s")
	}
`, tempStr, strValue, paramName, tempStr2, strValue)
		}

		if strValue, found := validators["max"]; found {
			fmt.Fprintf(out, `
	if p.%s > %s {
		return fmt.Errorf("%s must be <= %s")
	}
`, fieldName, strValue, paramName, strValue)
		}
	}

	fmt.Fprintln(out, "\treturn\n}")
}

type PathToHandler struct {
	Path        string
	HandlerName string
}

var paths = make(map[string][]PathToHandler)

type ApiGenSpecs struct {
	Url    string `json:"url"`
	Auth   bool   `json:"auth"`
	Method string `json:"method"`
}

func processFuncDecl(out io.Writer, f *ast.FuncDecl) {
	if f.Doc == nil {
		fmt.Printf("SKIP method %#v doesnt have comments\n", f.Name.Name)
		return
	}

	var specs ApiGenSpecs
	needCodegen := false
	prefix := "// apigen:api "
	for _, comment := range f.Doc.List {
		if strings.HasPrefix(comment.Text, prefix) {
			needCodegen = true
			err := json.Unmarshal([]byte(comment.Text[len(prefix):]), &specs)
			if err != nil {
				log.Fatal(err)
			}
			break
		}
	}
	if !needCodegen {
		fmt.Printf("SKIP struct %#v doesnt have apigen:api mark\n", f.Name.Name)
		return
	}

	var structName string
	handlerName := f.Name.Name
	for _, l := range f.Recv.List {
		structName = getTypeName(l.Type)
	}
	paths[structName] = append(paths[structName], PathToHandler{Path: specs.Url, HandlerName: "wrapper" + handlerName})

	paramsType := getTypeName(f.Type.Params.List[1].Type)
	genHandler(out, structName, handlerName, paramsType, &specs)
}

func genHandler(out io.Writer, structName string, handlerName string, paramsType string, specs *ApiGenSpecs) {
	fmt.Fprintf(out, "func (srv %s) wrapper%s(w http.ResponseWriter, r *http.Request) {\n", structName, handlerName)
	if specs.Method == "GET" || specs.Method == "POST" {
		fmt.Fprintf(out, `
	if r.Method != %q {
		w.WriteHeader(http.StatusNotAcceptable)
		writeResponse(w, "bad method", nil)
		return
	}
`, specs.Method)
	}
	if specs.Auth {
		fmt.Fprintln(out, `
	if r.Header.Get("X-Auth") != "100500" {
		w.WriteHeader(http.StatusForbidden)
		writeResponse(w, "unauthorized", nil)
		return
	}`)
	}
	fmt.Fprintf(out, `
	r.ParseForm()
	var params %s
	err := params.fillAndValidate(r.Form)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		writeResponse(w, err.Error(), nil)
		return
	}

	res, err := srv.%s(r.Context(), params)
	if err != nil {
		var apiError ApiError
		if errors.As(err, &apiError) {
			w.WriteHeader(apiError.HTTPStatus)
			writeResponse(w, apiError.Error(), nil)
		} else {
			w.WriteHeader(http.StatusInternalServerError)
			writeResponse(w, err.Error(), nil)
		}
		return
	}

	writeResponse(w, "", res)
}
`, paramsType, handlerName)
}

func genServeHTTP(out io.Writer) {
	for structName, wrappers := range paths {
		fmt.Fprintf(out, "func (h %s) ServeHTTP(w http.ResponseWriter, r *http.Request) {\n", structName)
		fmt.Fprintf(out, "\tswitch r.URL.Path {\n")
		for _, wrapper := range wrappers {
			fmt.Fprintf(out, `	case %q:
		h.%s(w, r)
`, wrapper.Path, wrapper.HandlerName)
		}
		fmt.Fprintf(out, `	default:
		w.WriteHeader(http.StatusNotFound)
		writeResponse(w, "unknown method", nil)
	}
}
`)
	}
}
