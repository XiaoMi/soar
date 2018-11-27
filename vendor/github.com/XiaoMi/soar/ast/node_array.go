/*
 * Copyright 2018 Xiaomi, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package ast

import (
	"errors"

	"github.com/XiaoMi/soar/common"
	"vitess.io/vitess/go/vt/sqlparser"
)

// 该文件用于构造一个存储AST生成节点的链表
// 以能够更好的对AST中的每个节点进行查询、跳转、重建等

// NodeItem 链表节点
type NodeItem struct {
	ID    int               // NodeItem 在 List 中的编号，与顺序有关
	Prev  *NodeItem         // 前一个节点
	Self  sqlparser.SQLNode // 自身指向的AST Node
	Next  *NodeItem         // 后一个节点
	Array *NodeList         // 指针指向所在的链表，用于快速跳转node
}

// NodeList 链表结构体
type NodeList struct {
	Length  int
	Head    *NodeItem
	NodeMap map[int]*NodeItem
}

// NewNodeList 从抽象语法树中构造一个链表
func NewNodeList(statement sqlparser.Statement) *NodeList {
	// 将AST构造成链表
	l := &NodeList{NodeMap: make(map[int]*NodeItem)}
	err := sqlparser.Walk(func(node sqlparser.SQLNode) (kontinue bool, err error) {
		l.Add(node)
		return true, nil
	}, statement)
	common.LogIfWarn(err, "")
	return l
}

// Add 将会把一个sqlparser.SQLNode添加到节点中
func (l *NodeList) Add(node sqlparser.SQLNode) *NodeItem {
	if l.Length == 0 {
		l.Head = &NodeItem{
			ID:    0,
			Self:  node,
			Next:  nil,
			Prev:  nil,
			Array: l,
		}
		l.NodeMap[l.Length] = l.Head
	} else {
		if n, ok := l.NodeMap[l.Length-1]; ok {
			n.Next = &NodeItem{
				ID:    l.Length - 1,
				Prev:  n,
				Self:  node,
				Next:  nil,
				Array: l,
			}
			l.NodeMap[l.Length] = n.Next
		}
	}
	l.Length++

	return l.NodeMap[l.Length-1]
}

// Remove 从链表中移除一个节点
func (l *NodeList) Remove(node *NodeItem) error {
	var err error
	defer func() {
		err := recover()
		if err != nil {
			common.Log.Error("func (l *NodeList) Remove recovered: %v", err)
		}
	}()

	if node.Array != l {
		return errors.New("node not belong to this array")
	}

	if node.Prev == nil {
		// 如果是头结点
		node.Next.Prev = nil
	} else if node.Next == nil {
		// 如果是尾节点
		node.Prev.Next = nil
	} else {
		// 删除节点，连接断开的链表
		node.Prev.Next = node.Next
		node.Next.Prev = node.Prev
		delete(l.NodeMap, node.ID)
	}

	return err
}

// First 返回链表头结点
func (l *NodeList) First() *NodeItem {
	return l.Head
}

// Last 返回链表末尾节点
func (l *NodeList) Last() *NodeItem {
	return l.NodeMap[l.Length-1]
}
