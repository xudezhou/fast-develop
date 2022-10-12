package main

import (
	"bufio"
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

	if len(os.Args) != 3 {
		fmt.Println("参数错误")
		return
	}

	files, _ := ioutil.ReadDir(os.Args[1])
	for _, f := range files {

		defer glog.CatchException()

		if strings.Contains(f.Name(), ".h") {
			if !strings.Contains(f.Name(), "table") {
				//continue
			}
			fmt.Println(f.Name())
			counts := make(map[int]string)
			var count int
			data, err := ioutil.ReadFile(os.Args[1] + "/" + f.Name())
			if err != nil {
				fmt.Fprintf(os.Stderr, "dup3: %v\n", err)
				continue
			}
			for _, line := range strings.Split(string(data), "\n") {
				line = strings.ReplaceAll(line, "\t", " ")
				counts[count] = line
				count++
			}
			write(search(counts), os.Args[2]+"/"+f.Name()[:len(f.Name())-2])
		}
	}
}

func search(fileMap map[int]string) []string {

	var index int

	var res []string

	for index < len(fileMap) {
		file := fileMap[index]
		temp := strings.ReplaceAll(file, "\t", " ")
		if len(temp) > 2 && temp[0:2] == "//" {
			res = append(res, file)
			index++
		} else {
			switch {
			case strings.Contains(file, "#define"):
				index = isDefine(fileMap, index)
			case strings.Contains(file, "enum"):
				index = isEnum(fileMap, index+2)
			case strings.Contains(file, "struct"), strings.Contains(file, "class"):
				if !strings.Contains(file, ";") {
					index = isType(fileMap, index, &res)
				} else {
					index++
				}
			default:
				index++
			}
		}
	}

	if len(enumList) > 0 {
		res = append(res, "const(")
		for _, enum := range enumList {
			res = append(res, fmt.Sprintf("%s = %s %s", enum[0], enum[1], enum[2]))
		}
		res = append(res, ")")
		enumList = [][3]string{}
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
		}
		iotaListList = []IotaStru{}
	}

	return res
}

func isType(fileMap map[int]string, index int, res *[]string) int {

	var temp []int
	index = dfs(fileMap, index, &temp)
	if temp == nil {
		return index
	}
	var nameList string

	if len(fileMap[temp[0]]) > 7 {
		nameList = strings.ReplaceAll(fileMap[temp[0]], "{", "")
	} else {
		if len(fileMap[temp[len(temp)-1]]) > 3 {
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
	*res = append(*res, fmt.Sprintf("type %s struct{", strings.ReplaceAll(name, "\r", "")))

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
		if strings.Contains(file, "(") {
			var t []int
			dfs2(fileMap, i, &t)
			i = t[1]
			continue
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

		if flag2 {
			type_ = "*" + type_
		}
		if name == "type" {
			name = "type_"
			*res = append(*res, fmt.Sprintf("%s %s  `json:\"type\"` %s", name, type_, annotation))
		} else {
			*res = append(*res, fmt.Sprintf("%s %s %s", name, type_, annotation))
		}
	}
	*res = append(*res, "}")
	return temp[len(temp)-1] + 1
}

var typeMap = map[string]string{
	"VenunInfo":  "xutil.VenunInfo",
	"XtCard":     "xutil.XtCard",
	"XtTypeDeck": "xutil.XtTypeDeck",
}

var enumList [][3]string

func isDefine(fileList map[int]string, index int) int {
	l, annotation, _ := toList(fileList[index])
	if len(l) == 3 {
		enumList = append(enumList, [3]string{l[1], l[2], annotation})
	}
	return index + 1
}

var iotaListList []IotaStru

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
		iotaStru.newTypeName = strings.ReplaceAll(iotaStru.newTypeName, "\r", "")
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

	line = strings.ReplaceAll(line, "\r", "")

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

func typeJudge(argStr string) string {

	argStr = strings.ReplaceAll(argStr, " ", "")

	if len(argStr) == 0 {
		return ""
	}

	var isCompoundType bool
	if strings.Contains(argStr, "map") || strings.Contains(argStr, "vector") {
		isCompoundType = true
	}

	switch {

	case strings.Contains(argStr, "int") && !isCompoundType, strings.Contains(argStr, "INT64") && !isCompoundType:
		return "int"
	case strings.Contains(argStr, "string") && !isCompoundType:
		return "string"
	case strings.Contains(argStr, "Json"):
		return "*simplejson.Json"
	case strings.Contains(argStr, "float") && !isCompoundType:
		return "float64"
	case strings.Contains(argStr, "bool") && !isCompoundType:
		return "bool"
	case strings.Contains(argStr, "map"):
		firstIndex := strings.IndexRune(argStr, '<')
		secondIndex := strings.IndexRune(argStr, ',')
		lastIndex := strings.LastIndexByte(argStr, '>')
		return fmt.Sprintf("map[%s]%s", typeJudge(argStr[firstIndex+1:secondIndex]), typeJudge(argStr[secondIndex+1:lastIndex]))
	case strings.Contains(argStr, "vector"):
		{
			firstIndex := strings.IndexRune(argStr, '<')
			lastIndex := strings.LastIndexByte(argStr, '>')
			return "[]" + typeJudge(argStr[firstIndex+1:lastIndex])
		}
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

func dfs(file map[int]string, start int, res *[]int) int {

	for {
		if start > len(file) {
			return 0
		}
		if strings.ContainsRune(file[start], '{') {
			*res = append(*res, start)
			start = dfs(file, start+1, res)
			if len(*res)%2 == 0 {
				return start + 1
			}
			continue
		} else if strings.ContainsRune(file[start], '}') {
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
