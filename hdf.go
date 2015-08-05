/*
 * Copyright (c) 2013 Patrick Wildt <patrick@blueri.se>
 *
 * Permission to use, copy, modify, and distribute this software for any
 * purpose with or without fee is hereby granted, provided that the above
 * copyright notice and this permission notice appear in all copies.
 *
 * THE SOFTWARE IS PROVIDED "AS IS" AND THE AUTHOR DISCLAIMS ALL WARRANTIES
 * WITH REGARD TO THIS SOFTWARE INCLUDING ALL IMPLIED WARRANTIES OF
 * MERCHANTABILITY AND FITNESS. IN NO EVENT SHALL THE AUTHOR BE LIABLE FOR
 * ANY SPECIAL, DIRECT, INDIRECT, OR CONSEQUENTIAL DAMAGES OR ANY DAMAGES
 * WHATSOEVER RESULTING FROM LOSS OF USE, DATA OR PROFITS, WHETHER IN AN
 * ACTION OF CONTRACT, NEGLIGENCE OR OTHER TORTIOUS ACTION, ARISING OUT OF
 * OR IN CONNECTION WITH THE USE OR PERFORMANCE OF THIS SOFTWARE.
 */

// Package hdf implements a simple library for interacting with
// ClearSilver's Hierarchical Data Format (HDF).
package hdf

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"
)

const (
	hdfRegexEqual      = "^\\s*([a-zA-Z0-9_.]+)\\s*=\\s*(.*)$"
	hdfRegexLink       = "^\\s*([a-zA-Z0-9_.]+)\\s*:\\s*(.*)$"
	hdfRegexOpenPaste  = "^\\s*([a-zA-Z0-9_.]+)\\s*<<\\s*EOM\\s*$"
	hdfRegexClosePaste = "^EOM$"
	hdfRegexOpenTree   = "^\\s*([a-zA-Z0-9_.]+)\\s*{\\s*$"
	hdfRegexCloseTree  = "^\\s*}\\s*$"
)

var debug *bool = flag.Bool("debug", false, "enable debug logging")

func Debugf(format string) {
	if *debug {
		log.Print("DEBUG " + format)
	}
}

func Errorf(format string) {
	log.Printf("ERROR " + format)
}

type HDF struct {
	Name     string
	Next     *HDF
	Parent   *HDF
	Children *HDF
	Link     string
	Value    string
}

func New() *HDF {
	return new(HDF)
}

// Parse parses an HDF file.
func (hdf *HDF) Parse(path string) {
	file, err := os.Open(path)
	if err != nil {
		fmt.Printf("File could not be opened.\n")
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for hdf.parseLine(scanner) {
	}
}

func (hdf *HDF) parseLine(scanner *bufio.Scanner) bool {
	if !scanner.Scan() {
		return false
	}
	s := scanner.Text()

	if m, _ := regexp.MatchString(hdfRegexEqual, s); m {
		ret := regexp.MustCompile(hdfRegexEqual).FindStringSubmatch(s)
		hdf.SetValue(ret[1], ret[2])
		Debugf(fmt.Sprintf("equals matched: %s = %s\n", ret[1], ret[2]))
	} else if m, _ := regexp.MatchString(hdfRegexLink, s); m {
		ret := regexp.MustCompile(hdfRegexLink).FindStringSubmatch(s)
		hdf.LinkValue(ret[1], ret[2])
		Debugf(fmt.Sprintf("link matched: %s : %s\n", ret[1], ret[2]))
	} else if m, _ := regexp.MatchString(hdfRegexOpenPaste, s); m {
		ret := regexp.MustCompile(hdfRegexOpenPaste).FindStringSubmatch(s)
		Debugf(fmt.Sprintf("EOM begin matched: %s\n", ret[1]))
		var val string
		for {
			scanner.Scan()
			s = scanner.Text()
			Debugf(fmt.Sprintf("%s\n", s))
			if m, _ := regexp.MatchString(hdfRegexClosePaste, s); m {
				Debugf("EOM end\n")
				break
			}
			val += s + "\n"
		}
		hdf.SetValue(ret[1], val)
	} else if m, _ := regexp.MatchString(hdfRegexOpenTree, s); m {
		ret := regexp.MustCompile(hdfRegexOpenTree).FindStringSubmatch(s)
		hdf = hdf.getObjectByPathOrCreate(ret[1])
		for hdf.parseLine(scanner) {
		}
		Debugf(fmt.Sprintf("open tree matched: %s\n", ret[1]))
	} else if m, _ := regexp.MatchString(hdfRegexCloseTree, s); m {
		Debugf("close tree matched\n")
		return false
	} else {
		Debugf(fmt.Sprintf("%s\n", s))
		Debugf("nothing matched\n")
	}

	return true
}

// GetObject retrieves an object identified by 'path'.
// It returns nil if the object doesn't exist.
func (hdf *HDF) GetObject(path string) *HDF {
	return hdf.getObjectByPath(path)
}

// GetValue retrieves the value of an object identified by 'path' as string.
// It returns the passed alternative string, if the object or value
// does not exist.
func (hdf *HDF) GetValue(path string, alt string) string {
	obj := hdf.getObjectByPath(path)
	if obj != nil && len(obj.Value) != 0 {
		return obj.Value
	} else {
		return alt
	}
}

// GetIntValue retrieves the value of an object identified by 'path' as integer.
// It returns the passed alternative integer, if the object or value
// does not exist.
func (hdf *HDF) GetIntValue(path string, alt int) int {
	s := hdf.GetValue(path, "")
	if len(s) != 0 {
		i, err := strconv.Atoi(s)
		if err == nil {
			return i
		}
	}
	return alt
}

// SetValue sets the value of an object identified by 'path' as string.
// It creates the object if it doesn't exist.
func (hdf *HDF) SetValue(path string, value string) {
	obj := hdf.getObjectByPathOrCreate(path)
	obj.Value = value
}

// SetIntValue sets the value of an object identified by 'path' as integer.
// It creates the object if it doesn't exist.
func (hdf *HDF) SetIntValue(path string, value int) {
	hdf.SetValue(path, strconv.Itoa(value))
}

// DeleteValue deletes the value of an object, identified by 'path',
// if it exists.
func (hdf *HDF) DeleteValue(path string) {
	obj := hdf.getObjectByPath(path)
	if obj != nil {
		obj.Parent.deleteObject(obj)
	}
}

// Link links one tree to another
func (hdf *HDF) LinkValue(from string, to string) {
	obj := hdf.getObjectByPathOrCreate(from)
	obj.Children = nil
	obj.Value = ""
	obj.Link = to
}

// ObjectName returns the Name of the object.
func (hdf *HDF) ObjectName() string {
	return hdf.Name
}

// addObject inserts an Object as its child.
func (hdf *HDF) addObject(node *HDF) {
	node.Parent = hdf
	if hdf.Children == nil {
		node.Next = hdf.Children
		hdf.Children = node
	} else {
		if node.Name < hdf.Children.Name {
			node.Next = hdf.Children
			hdf.Children = node
		} else {
			child := hdf.Children
			for ; child.Next != nil && child.Next.Name < node.Name; child = child.Next {
				/* empty body */
			}
			node.Next = child.Next
			child.Next = node
		}
	}
}

// deleteObject deletes an object.
// If the first child is the node, set the child pointer
// to the next child.  If it was only one child, delete the parent.
// Else, go through all children and look for the child
// Remove it from the list.
func (hdf *HDF) deleteObject(node *HDF) {
	if hdf.Children == node {
		hdf.Children = node.Next
		if hdf.Children == nil {
			hdf.Parent.deleteObject(hdf)
		}
		return
	}

	for child := hdf.Children; child.Next != nil; child = child.Next {
		if child.Next == node {
			child.Next = node.Next
			return
		}
	}
}

// getObjects iterates through all objects on the same level
// to search for a specific object identified by 's'.
func (hdf *HDF) getObject(s string) *HDF {
	for node := hdf; node != nil; node = node.Next {
		if node.Name == s {
			if len(node.Link) != 0 {
				node = node.getRoot().getObjectByPathOrCreate(node.Link)
			}
			return node
		}
	}
	return nil
}

// getObjectByPath splits the path and try to follow the path
// down to the object.  It return the object.
func (hdf *HDF) getObjectByPath(s string) *HDF {
	ss := splitPath(s)

	for _, val := range ss {
		hdf = hdf.Children
		obj := hdf.getObject(val)
		if obj == nil {
			return nil
		}
		hdf = obj
	}
	return hdf
}

// createObject creates an object and links it to his parent.
func (hdf *HDF) createObject(s string) *HDF {
	var obj HDF
	obj.Name = s
	hdf.addObject(&obj)
	return &obj
}

// createObjectByPath creates an object and all objects needed in
// tree.
func (hdf *HDF) createObjectByPath(path string) *HDF {
	ss := splitPath(path)
	var obj *HDF = nil
	for _, val := range ss {
		obj = hdf.Children.getObject(val)
		if obj == nil {
			obj = hdf.createObject(val)
		}
		hdf = obj
	}
	if obj == nil {
		fmt.Println("obj not created")
	}
	return obj
}

// getObjectByPathOrCreate retreives an Object identified by 'path'.
// If it doesn't exist, it create it.
func (hdf *HDF) getObjectByPathOrCreate(path string) *HDF {
	obj := hdf.getObjectByPath(path)
	if obj == nil {
		obj = hdf.createObjectByPath(path)
	}
	return obj
}

// getRoot follows the tree backwards to its root element
func (hdf *HDF) getRoot() *HDF {
	for hdf.Parent != nil {
		hdf = hdf.Parent
	}
	return hdf
}

// hasChildByObject checks that a node has a child identified by an object reference.
func (hdf *HDF) hasChildByObject(obj *HDF) bool {
	exist := false
	for node := hdf.Children; node != nil; node = node.Next {
		if node == obj {
			exist = true
			break
		}
	}
	return exist
}

// hasChildByName checks that a node has a child identified by 's'.
func (hdf *HDF) hasChildByName(s string) bool {
	exist := false
	for node := hdf.Children; node != nil; node = node.Next {
		if node.Name == s {
			exist = true
			break
		}
	}
	return exist
}

// splitPath splits a string by ".".
func splitPath(path string) []string {
	return strings.Split(path, ".")
}

// DumpTree dumps the tree in a hierarchical format.
func (hdf *HDF) DumpTree() []string {
	return dumpTree(hdf.Children, 0, nil)
}

// DumpFlat dumps the tree in a flat format.
func (hdf *HDF) DumpFlat() []string {
	dump := dumpFlat(hdf.Children, "", nil)
	return dump
}

func dumpFlat(hdf *HDF, s string, _dump []string) []string {
	var dump []string
	if _dump != nil {
		dump = _dump
	}

	for child := hdf; child != nil; child = child.Next {
		var n string = s
		if len(s) != 0 {
			n = n + "."
		}
		n = n + child.Name
		if len(child.Value) != 0 {
			if strings.Contains(child.Value, "\n") {
				dump = append(dump, fmt.Sprintf("%s << EOM\n%sEOM", n, child.Value))
			} else {
				dump = append(dump, fmt.Sprintf("%s = %s", n, child.Value))
			}
		}
		if len(child.Link) != 0 {
			dump = append(dump, fmt.Sprintf("%s : %s", n, child.Link))
		}
		if child.Children != nil {
			dump = dumpFlat(child.Children, n, dump)
		}
	}
	return dump
}

func dumpTree(hdf *HDF, w int, _dump []string) []string {
	var dump []string
	if _dump != nil {
		dump = _dump
	}

	var padding string
	for i := 0; i < w; i++ {
		padding += "\t"
	}

	for child := hdf; child != nil; child = child.Next {
		if len(child.Value) != 0 {
			if strings.Contains(child.Value, "\n") {
				dump = append(dump, fmt.Sprintf("%s%s << EOM\n%sEOM", padding, child.Name, child.Value))
			} else {
				dump = append(dump, fmt.Sprintf("%s%s = %s", padding, child.Name, child.Value))
			}
		}
		if len(child.Link) != 0 {
			dump = append(dump, fmt.Sprintf("%s%s : %s", padding, child.Name, child.Link))
		}
		if child.Children != nil {
			dump = append(dump, fmt.Sprintf("%s%s {", padding, child.Name))
			dump = dumpTree(child.Children, w+1, dump)
			dump = append(dump, fmt.Sprintf("%s}", padding))
		}
	}

	return dump
}
