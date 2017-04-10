package printer

import (
	"fmt"

	"github.com/davyxu/tabtoy/exportorv2/i18n"
	"github.com/davyxu/tabtoy/exportorv2/model"
	"github.com/davyxu/tabtoy/util"
)

func valueWrapperLua(t model.FieldType, n *model.Node) string {

	switch t {
	case model.FieldType_String:
		return util.StringEscape(n.Value)
	case model.FieldType_Enum:
		return fmt.Sprintf("\"%s\"", n.Value)
	}

	return n.Value
}

type luaPrinter struct {
}

func (self *luaPrinter) Run(g *Globals) *Stream {

	bf := NewStream()

	bf.Printf("-- Generated by github.com/davyxu/tabtoy\n")
	bf.Printf("-- Version: %s\n", g.Version)

	bf.Printf("\nlocal tab = {\n")

	for tabIndex, tab := range g.Tables {

		if !tab.LocalFD.MatchTag(".lua") {
			log.Infof("%s: %s", i18n.String(i18n.Printer_IgnoredByOutputTag), tab.Name())
			continue
		}

		if !printTableLua(bf, tab) {
			return nil
		}

		// 根字段分割
		if tabIndex < len(g.Tables)-1 {
			bf.Printf(", ")
		}

		bf.Printf("\n\n")
	}

	// local tab = {
	bf.Printf("}\n\n")

	if !genLuaIndexCode(bf, g.CombineStruct) {
		return bf
	}

	// 生成枚举
	if !genLuaEnumCode(bf, g.FileDescriptor) {
		return bf
	}

	bf.Printf("\nreturn tab")

	return bf
}

func printTableLua(bf *Stream, tab *model.Table) bool {

	bf.Printf("	%s = {\n", tab.LocalFD.Name)

	// 遍历每一行
	for rIndex, r := range tab.Recs {

		// 每一行开始
		bf.Printf("		{ ")

		// 遍历每一列
		for rootFieldIndex, node := range r.Nodes {

			if node.IsRepeated {
				bf.Printf("%s = { ", node.Name)
			} else {
				bf.Printf("%s = ", node.Name)
			}

			// 普通值
			if node.Type != model.FieldType_Struct {

				if node.IsRepeated {

					// repeated 值序列
					for arrIndex, valueNode := range node.Child {

						bf.Printf("%s", valueWrapperLua(node.Type, valueNode))

						// 多个值分割
						if arrIndex < len(node.Child)-1 {
							bf.Printf(", ")
						}

					}
				} else {
					// 单值
					valueNode := node.Child[0]

					bf.Printf("%s", valueWrapperLua(node.Type, valueNode))

				}

			} else {

				// 遍历repeated的结构体
				for structIndex, structNode := range node.Child {

					// 结构体开始
					bf.Printf("{ ")

					// 遍历一个结构体的字段
					for structFieldIndex, fieldNode := range structNode.Child {

						// 值节点总是在第一个
						valueNode := fieldNode.Child[0]

						bf.Printf("%s= %s", fieldNode.Name, valueWrapperLua(fieldNode.Type, valueNode))

						// 结构体字段分割
						if structFieldIndex < len(structNode.Child)-1 {
							bf.Printf(", ")
						}

					}

					// 结构体结束
					bf.Printf(" }")

					// 多个结构体分割
					if structIndex < len(node.Child)-1 {
						bf.Printf(", ")
					}

				}

			}

			if node.IsRepeated {
				bf.Printf(" }")
			}

			// 根字段分割
			if rootFieldIndex < len(r.Nodes)-1 {
				bf.Printf(", ")
			}

		}

		// 每一行结束
		bf.Printf(" 	}")

		if rIndex < len(tab.Recs)-1 {
			bf.Printf(",")
		}

		bf.Printf("\n")

	}

	// Sample = {
	bf.Printf("	}")

	return true

}

// 收集需要构建的索引的类型
func genLuaEnumCode(bf *Stream, globalFile *model.FileDescriptor) bool {

	bf.Printf("\ntab.Enum = {\n")

	// 遍历字段
	for _, d := range globalFile.Descriptors {

		if d.Kind != model.DescriptorKind_Enum {
			continue
		}

		bf.Printf("	%s = {\n", d.Name)

		for _, fd := range d.Fields {
			bf.Printf("		[\"%s\"] = %d,\n", fd.Name, fd.EnumValue)
		}

		bf.Printf("	},\n")

	}

	bf.Printf("}\n")

	return true

}

// 收集需要构建的索引的类型
func genLuaIndexCode(bf *Stream, combineStruct *model.Descriptor) bool {

	// 遍历字段
	for _, fd := range combineStruct.Fields {

		// 这个字段被限制输出
		if fd.Complex != nil && !fd.Complex.File.MatchTag(".lua") {
			continue
		}

		// 对CombineStruct的XXDefine对应的字段
		if combineStruct.Usage == model.DescriptorUsage_CombineStruct {

			// 这个结构有索引才创建
			if fd.Complex != nil && len(fd.Complex.Indexes) > 0 {

				// 索引字段
				for _, key := range fd.Complex.Indexes {
					mapperVarName := fmt.Sprintf("tab.%sBy%s", fd.Name, key.Name)

					bf.Printf("\n-- %s\n", key.Name)
					bf.Printf("%s = {}\n", mapperVarName)
					bf.Printf("for _, rec in pairs(tab.%s) do\n", fd.Name)
					bf.Printf("\t%s[rec.%s] = rec\n", mapperVarName, key.Name)
					bf.Printf("end\n")
				}

			}

		}

	}

	return true

}

func init() {

	RegisterPrinter("lua", &luaPrinter{})

}
