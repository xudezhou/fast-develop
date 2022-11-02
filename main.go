package main

import (
	"bufio"
	"flag"
	"fmt"
	"github.com/gocpp/glog"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
)

func write(typeList []string, outputName string) {

	filePath := outputName + "_struct.go"

	os.Remove(filePath)
	file, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		fmt.Println("文件打开失败", err)
	}
	//及时关闭file句柄
	defer file.Close()
	//写入文件时，使用带缓存的 *Writer
	w := bufio.NewWriter(file)
	w.WriteString("package main\n")
	for i := 0; i < len(typeList); i++ {
		w.WriteString(typeList[i] + "\n")
	}
	//Flush将缓存的文件真正写入到文件中
	w.Flush()

}

func main() {
	c2g()
}

type Param struct {
	inputPath  string
	outputPath string
	json       bool
	hump       bool
}

var p Param

func init() {
	flag.StringVar(&p.inputPath, "i", "", "a")
	flag.StringVar(&p.outputPath, "o", "", "a")
	flag.BoolVar(&p.json, "j", false, "a")
	flag.BoolVar(&p.hump, "h", false, "a")
	flag.Parse()
}

func c2g() {

	if p.inputPath == "" || p.outputPath == "" {
		glog.Fatalln("inputPath or outputPath is empty")
	}

	files, _ := ioutil.ReadDir(p.inputPath)
	for _, f := range files {

		defer glog.CatchException()

		if strings.Contains(f.Name(), ".h") {
			if !strings.Contains(f.Name(), "table") {
				//continue
			}
			fmt.Println(f.Name())
			counts := make(map[int]string)
			var count int
			data, err := ioutil.ReadFile(p.inputPath + "/" + f.Name())
			if err != nil {
				fmt.Fprintf(os.Stderr, "dup3: %v\n", err)
				continue
			}
			for _, line := range strings.Split(string(data), "\n") {
				line = strings.ReplaceAll(line, "\t", " ")
				line = strings.ReplaceAll(line, "\r", " ")
				counts[count] = line
				count++
			}
			write(search(counts), p.outputPath+"/"+f.Name()[:len(f.Name())-2])
		}
	}
}

func search(fileMap map[int]string) []string {

	var index int

	var res []string

	for index < len(fileMap) {
		file := fileMap[index]

		hasAnotation, nextIndex := filterAnotation(fileMap, index)

		if hasAnotation {
			for i := index; i < nextIndex; i++ {
				if strings.Contains(fileMap[i], "if") {
					continue
				}
				res = append(res, fileMap[i])
			}
			file = fileMap[nextIndex]
			index = nextIndex
		}

		switch {
		case strings.Contains(file, "#define"):
			index = isDefine(fileMap, index)
		case strings.Contains(file, "enum"):
			index = isEnum(fileMap, index+2)
		case (strings.Contains(file, "struct") || strings.Contains(file, "class")) && !strings.Contains(file, "("):
			if !strings.Contains(file, ";") {
				index = isType(fileMap, index)
			} else {
				index++
			}
		default:
			index++
		}

	}

	if len(enumList) > 0 {
		res = append(res, "const(")
		for _, enum := range enumList {
			res = append(res, fmt.Sprintf("%s = %s %s", enum[0], enum[1], enum[2]))
		}
		res = append(res, ")")
		enumList = [][3]string{}
		res = append(res, "")
	}

	if len(iotaListList) > 0 {
		for _, iotaStru := range iotaListList {
			if iotaStru.isNewType {
				res = append(res, fmt.Sprintf("type %s int", iotaStru.newTypeName))
			}
			res = append(res, "const(")
			for i, enum := range iotaStru.iotaList {
				if i == 0 {
					var startNum int
					strconv.ParseInt(enum[1], 10, 64)

					if startNum != 0 {
						res = append(res, fmt.Sprintf("%s %s= iota + %d %s", enum[0], iotaStru.newTypeName, startNum, enum[2]))
					} else {
						res = append(res, fmt.Sprintf("%s %s= iota  %s", enum[0], iotaStru.newTypeName, enum[2]))
					}
				} else {
					res = append(res, fmt.Sprintf("%s  %s", strings.ReplaceAll(enum[0], ";", ""), enum[2]))
				}
			}
			res = append(res, ")")
			res = append(res, "")
		}
		iotaListList = []IotaStru{}
	}

	if len(typeList) > 0 {
		for _, typeStruct := range typeList {

			res = append(res, fmt.Sprintf("type %s struct {", typeStruct.structName))

			for _, field := range typeStruct.attributeList {

				res = append(res, fmt.Sprintf("%s %s %s", field[0], field[1], field[2]))
			}
			res = append(res, "}")
			res = append(res, "")
		}
		typeList = []TypeStruct{}
	}

	return res
}

func filterAnotation(fileMap map[int]string, index int) (hasAnotation bool, nextLine int) {

	nextLine = index
	var anotations bool
	for {
		temp, ok := fileMap[nextLine]
		if !ok {
			break
		}
		temp = strings.ReplaceAll(temp, " ", "")
		if anotations {
			nextLine++
			if len(temp) >= 2 && strings.Contains(temp, "*/") {
				anotations = false
				continue
			}
			continue
		}
		if temp == "" {
			hasAnotation = true
			nextLine++
			continue
		}
		if strings.Contains(temp, "if") {
			hasAnotation = true
			nextLine++
			continue
		}
		if len(temp) >= 2 && temp[0:2] == "//" {
			hasAnotation = true
			nextLine++
			continue
		}
		if len(temp) >= 2 && temp[0:2] == "/*" {
			nextLine++
			if len(temp) >= 3 && strings.Contains(temp, "*/") {
				continue
			}
			anotations = true
			hasAnotation = true
			continue
		}
		break
	}

	return
}

func isType(fileMap map[int]string, index int) int {

	var temp []int
	left, right = 0, 0
	index = dfs(fileMap, index, &temp)
	if temp == nil {
		return index
	}
	var nameList string

	t := strings.ReplaceAll(fileMap[temp[0]], " ", "")
	if len(t) > 7 {
		nameList = strings.ReplaceAll(fileMap[temp[0]], "{", "")
	} else {
		t = strings.ReplaceAll(fileMap[temp[len(temp)-1]], " ", "")
		if len(t) > 3 {
			nameList = fileMap[temp[len(temp)-1]]
		} else {
			nameList = fileMap[temp[0]-1]
		}
	}
	var name string
	if strings.Contains(nameList, "}") {
		name = strings.ReplaceAll(nameList, "}", "")
		name = strings.ReplaceAll(name, ";", "")
	} else {
		splitList := strings.Split(nameList, " ")
		for i := len(splitList) - 1; i >= 0; i-- {
			if splitList[i] != "" {
				name = splitList[i]
				break
			}
		}
	}
	var typeStruct TypeStruct
	typeStruct.structName = name
	var flag bool
	for i := temp[0] + 1; i < temp[len(temp)-1]; i++ {

		for j := 0; j < len(temp); j++ {
			if i == temp[j] && j+1 < len(temp) {
				i = temp[j+1]
				flag = true
				break
			}
		}

		if flag {
			flag = false
			continue
		}

		file := fileMap[i]

		hasAnotation, nextIndex := filterAnotation(fileMap, i)

		if hasAnotation {

			for j := i; j < nextIndex; j++ {
				if strings.Contains(fileMap[j], "if") {
					continue
				}
				typeStruct.attributeList = append(typeStruct.attributeList, [3]string{"", "", fileMap[j]})
			}

			file = fileMap[nextIndex]
			i = nextIndex
		}

		if strings.Contains(file, "struct") && !strings.Contains(file, "(") {
			i = isType(fileMap, i)
			file = fileMap[i]
		}

		if strings.Contains(file, "enum") {
			i = isEnum(fileMap, i+2)
			file = fileMap[i]
		}

		if strings.Contains(file, "(") {
			var t []int
			dfs2(fileMap, i, &t)
			i = t[1]
			continue
		}
		var oldAttribute string
		if strings.Contains(file, "=") {
			oldAttribute = "//" + file
			IndexRune := strings.IndexRune(file, '=')
			file = file[:IndexRune]
		}

		l, annotation, _ := toList(file)
		file = strings.ReplaceAll(file, annotation, "")

		if len(l) == 0 {
			continue
		}

		name = l[len(l)-1]

		name = strings.ReplaceAll(name, ";", "")

		nameIndex := strings.LastIndex(file, name)

		type_ := typeJudge(file[:nameIndex])
		if type_ == "" {
			continue
		}

		var flag2 bool
		if strings.Contains(name, "*") {
			flag2 = true
			name = name[1:]
		}
		if strings.Contains(name, "{") {

			firstIndex := strings.IndexRune(name, '{')

			if firstIndex != -1 {
				type_ = name[firstIndex:] + type_
				name = name[:firstIndex]
			}
		}

		var oldName = name

		if p.hump {

			for j := 0; j < len(name); j++ {
				if name[j] == '_' && j+1 < len(name) && name[j+1] >= 'a' && name[j+1] <= 'z' {
					name = name[:j+1] + string(name[j+1]-32) + name[j+2:]
				}
			}
			name = strings.ReplaceAll(name, "_", "")
		}

		if p.json {
			if len(name) > 0 && name[0] >= 'a' && name[0] <= 'z' {
				name = strings.ToUpper(name[:1]) + name[1:]
			}
		}

		if flag2 {
			type_ = "*" + type_
		}

		if name == "type" {
			name = "Type"
			typeStruct.attributeList = append(typeStruct.attributeList, [3]string{name, type_, fmt.Sprintf(" `json:\"type\"` %s %s", annotation, oldAttribute)})
		} else {
			if p.json {
				typeStruct.attributeList = append(typeStruct.attributeList, [3]string{name, type_, fmt.Sprintf(" `json:\"%s\"` %s %s", oldName, annotation, oldAttribute)})
			} else {
				typeStruct.attributeList = append(typeStruct.attributeList, [3]string{name, type_, fmt.Sprintf(" %s %s", annotation, oldAttribute)})
			}
		}
	}

	typeList = append(typeList, typeStruct)
	return temp[len(temp)-1] + 1
}

var typeMap = map[string]string{}

var enumList [][3]string

var iotaListList []IotaStru

var typeList []TypeStruct

type TypeStruct struct {
	structName    string
	attributeList [][3]string
}

func isDefine(fileList map[int]string, index int) int {
	l, annotation, _ := toList(fileList[index])
	if len(l) == 3 {
		enumList = append(enumList, [3]string{l[1], l[2], annotation})
	}
	return index + 1
}

type IotaStru struct {
	isNewType   bool
	newTypeName string
	iotaList    [][3]string
}

func isEnum(fileMap map[int]string, index int) int {

	var iotaStru IotaStru
	var tempEnumList [][3]string

	for !strings.Contains(fileMap[index], "}") {

		fileMap[index] = strings.ReplaceAll(fileMap[index], "=", "")
		l, annotation, _ := toList(fileMap[index])
		if len(l) == 0 {
			index++
			continue
		}
		if len(l) == 1 {
			iotaStru.iotaList = append(iotaStru.iotaList, [3]string{strings.ReplaceAll(l[0], ",", ""), "", annotation})
		} else {
			tempEnumList = append(tempEnumList, [3]string{l[0], strings.ReplaceAll(l[1], ",", ""), annotation})
		}
		index++
	}

	if len(fileMap[index]) > 3 {
		iotaStru.isNewType = true
		iotaStru.newTypeName = strings.ReplaceAll(fileMap[index], "}", "")
		iotaStru.newTypeName = strings.ReplaceAll(iotaStru.newTypeName, ";", "")
	}

	if len(iotaStru.iotaList) > 0 {
		if len(iotaStru.iotaList) > 0 {
			iotaStru.iotaList = append(tempEnumList, iotaStru.iotaList...)
			iotaListList = append(iotaListList, iotaStru)
		}
	} else {
		enumList = append(enumList, tempEnumList...)
	}
	return index + 1
}

func toList(line string) ([]string, string, string) {

	var strList []string
	var annotation string

	firstIndex := strings.IndexRune(line, '/')

	if firstIndex != -1 {
		annotation = line[firstIndex:]
		line = line[:firstIndex]
	}

	splitList := strings.Split(line, " ")

	for _, s := range splitList {
		if len(s) > 0 {
			strList = append(strList, s)
		}
	}

	return strList, annotation, line
}

const (
	isSet = iota
	isMap
	isSlice
)

func typeJudge(argStr string) string {

	argStr = strings.ReplaceAll(argStr, " ", "")

	if len(argStr) == 0 {
		return ""
	}

	var isCompoundType bool
	var dataType = len(argStr)

	if strings.Contains(argStr, "map") || strings.Contains(argStr, "vector") || strings.Contains(argStr, "set") {
		isCompoundType = true
		indexOfMap := strings.Index(argStr, "map")
		indexOfList := strings.Index(argStr, "vector")
		indexOfSet := strings.Index(argStr, "set")

		if indexOfMap != -1 && indexOfMap < dataType {
			dataType = isMap
		}
		if indexOfList != -1 && indexOfList < dataType {
			dataType = isSlice
		}
		if indexOfSet != -1 && indexOfSet < dataType {
			dataType = isSet
		}
	}

	switch {
	case strings.Contains(argStr, "int") && !isCompoundType, strings.Contains(argStr, "INT64") && !isCompoundType, strings.Contains(argStr, "long") && !isCompoundType:
		return "int"
	case strings.Contains(argStr, "string") && !isCompoundType:
		return "string"
	case strings.Contains(argStr, "Json"):
		return "*simplejson.Json"
	case strings.Contains(argStr, "float") && !isCompoundType:
		return "float64"
	case strings.Contains(argStr, "bool") && !isCompoundType:
		return "bool"
	case strings.Contains(argStr, "set") && dataType == isSet:
		firstIndex := strings.IndexRune(argStr, '<')
		lastIndex := strings.LastIndexByte(argStr, '>')
		return fmt.Sprintf("map[%s]struct{}", typeJudge(argStr[firstIndex+1:lastIndex]))
	case strings.Contains(argStr, "map") && dataType == isMap:
		firstIndex := strings.IndexRune(argStr, '<')
		secondIndex := strings.IndexRune(argStr, ',')
		lastIndex := strings.LastIndexByte(argStr, '>')
		return fmt.Sprintf("map[%s]%s", typeJudge(argStr[firstIndex+1:secondIndex]), typeJudge(argStr[secondIndex+1:lastIndex]))
	case strings.Contains(argStr, "vector") && dataType == isSlice:
		firstIndex := strings.IndexRune(argStr, '<')
		lastIndex := strings.LastIndexByte(argStr, '>')
		return "[]" + typeJudge(argStr[firstIndex+1:lastIndex])
	default:
		if strings.Contains(argStr, "*") {
			argStr = "*" + strings.ReplaceAll(argStr, "*", "")
		}
		if typeMap[argStr] != "" {
			return typeMap[argStr]
		}
		return argStr
	}

}

var left, right int

func dfs(fileMap map[int]string, start int, res *[]int) int {

	for {
		if start > len(fileMap) {
			return 0
		}
		hasAnotation, nextIndex := filterAnotation(fileMap, start)
		if hasAnotation {
			start = nextIndex
		}
		if strings.ContainsRune(fileMap[start], '{') {
			left++
			*res = append(*res, start)
			if strings.Contains(fileMap[start], "}") {
				start++
				right++
				continue
			}
			start = dfs(fileMap, start+1, res)
			if left == right {
				return start + 1
			}
			continue
		} else if strings.ContainsRune(fileMap[start], '}') {
			right++
			*res = append(*res, start)
			return start + 1
		}
		start++
	}
}

func dfs2(file map[int]string, start int, res *[]int) {

	for {
		if start > len(file) {
			return
		}
		if strings.ContainsRune(file[start], '(') {
			*res = append(*res, start)
		}
		if strings.ContainsRune(file[start], ')') {
			*res = append(*res, start)
			return
		}
		start++
	}
}
