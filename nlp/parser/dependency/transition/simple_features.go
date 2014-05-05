package transition

import (
// "log"
// nlp "chukuparser/nlp/types"
// "chukuparser/util"
// "math"
// "regexp"
// "sort"
// "strconv"
// "strings"
)

const (
	SET_SEPARATOR = "-"
)

var _Zpar_Bug_N1N2 bool = false

func (c *SimpleConfiguration) Address(location []byte, sourceOffset int) (int, bool, bool) {
	source := c.GetSource(location[0])
	if source == nil {
		return 0, false, false
	}
	atAddress, exists := source.Index(int(sourceOffset))
	if !exists {
		// zpar bug parity
		if _Zpar_Bug_N1N2 && location[0] == 'N' && c.Queue().Size() == 0 && sourceOffset > 0 {
			return sourceOffset - 1, true, false
		}
		// end zpar bug parity
		return 0, false, false
	}
	// test if feature address is a generator of feature (e.g. for each child..)
	locationLen := len(location)
	if locationLen >= 4 {
		if string(location[2:4]) == "Ci" {
			return atAddress, true, true
		}
	}

	location = location[2:]
	if len(location) == 0 {
		return atAddress, true, false
	}
	switch location[0] {
	case 'l', 'r':
		leftMods, rightMods := c.GetModifiers(atAddress)
		if location[0] == 'l' {
			if len(leftMods) == 0 {
				return 0, false, false
			}
			if len(location) > 1 && location[1] == '2' {
				if len(leftMods) > 1 {
					return leftMods[1], true, false
				}
			} else {
				return leftMods[0], true, false
			}
		} else {
			if len(rightMods) == 0 {
				return 0, false, false
			}
			if len(location) > 1 && location[1] == '2' {
				if len(rightMods) > 1 {
					return rightMods[len(rightMods)-2], true, false
				}
			} else {
				return rightMods[len(rightMods)-1], true, false
			}
		}
	case 'h':
		head, headExists := c.GetHead(atAddress)
		if headExists {
			if len(location) > 1 && location[1] == '2' {
				headOfHead, headOfHeadExists := c.GetHead(head.ID())
				if headOfHeadExists {
					return headOfHead.ID(), true, false
				}
			} else {
				return head.ID(), true, false
			}
		}
	}
	return 0, false, false
}

func (c *SimpleConfiguration) GenerateAddresses(nodeID int, location []byte) (nodeIDs []int) {
	if nodeID < 0 || nodeID >= len(c.Nodes) {
		return
	}
	if string(location[2:4]) == "Ci" {
		leftChildren, rightChildren := c.GetModifiers(nodeID)
		numLeft := len(leftChildren)
		nodeIDs = make([]int, numLeft+len(rightChildren))
		for i, leftChild := range leftChildren {
			nodeIDs[i] = leftChild
		}
		for j, rightChild := range rightChildren {
			nodeIDs[j+numLeft] = rightChild
		}
		return
	}
	return
}

func (c *SimpleConfiguration) GetModifierLabel(modifierID int) (int, bool) {
	arcs := c.Arcs().Get(&BasicDepArc{-1, -1, modifierID, ""})
	if len(arcs) > 0 {
		index, _ := c.ERel.IndexOf(arcs[0].GetRelation())
		return index, true
	}
	return 0, false
}

func (c *SimpleConfiguration) Attribute(source byte, nodeID int, attribute []byte) (interface{}, bool) {
	if nodeID < 0 || nodeID >= len(c.Nodes) {
		return 0, false
	}
	switch attribute[0] {
	case 'o':
		return c.NumHeadStack, true
	case 'd':
		return c.GetConfDistance()
	case 'w':
		if len(attribute) > 1 && attribute[1] == 'p' {
			node := c.GetRawNode(nodeID)
			return node.TokenPOS, true
		} else {
			node := c.GetRawNode(nodeID)
			return node.Token, true
		}
	case 'p':
		node := c.GetRawNode(nodeID)
		// TODO: CPOS
		return node.POS, true
	case 'l':
		//		relation, relExists :=
		return c.GetModifierLabel(nodeID)
	case 'v':
		if len(attribute) != 2 {
			return 0, false
		}
		leftMods, rightMods := c.GetNumModifiers(nodeID)
		switch attribute[1] {
		case 'l':
			return leftMods, true
		case 'r':
			return rightMods, true
		case 'f':
			return leftMods + rightMods, true
		}
	case 's':
		if len(attribute) != 2 {
			return 0, false
		}
		leftLabelSet, rightLabelSet, allLabels := c.GetModifierLabelSets(nodeID)
		switch attribute[1] {
		case 'l':
			return leftLabelSet, true
		case 'r':
			return rightLabelSet, true
		case 'f':
			return allLabels, true
		}
	case 'f':
		if len(attribute) == 2 && attribute[1] == 'p' {
			allModPOS := c.GetModifiersPOS(nodeID)
			return allModPOS, true
		}
	case 'm':
		if len(attribute) < 2 {
			return 0, false
		}
		node := c.GetRawNode(nodeID)
		switch attribute[2] {
		case 'h':
			if len(attribute) < 3 {
				return node.MHost, true
			}
			if len(attribute) > 3 {
				return 0, false
			}
			switch attribute[3] {
			case 'g':
				return node.MHostGen, true
			case 'n':
				return node.MHostNum, true
			case 'd':
				return node.MHostDef, true
			case 't':
				return node.MHostTense, true
			case 'p':
				return node.MHostPer, true
			}
		case 's':
			if len(attribute) < 3 {
				return node.MSuffix, true
			}
			if len(attribute) > 3 {
				return 0, false
			}
			switch attribute[3] {
			case 'g':
				return node.MSuffixGen, true
			case 'n':
				return node.MSuffixNum, true
			case 'p':
				return node.MSuffixDef, true
			}
		}
	}
	return 0, false
}

func (c *SimpleConfiguration) GetConfDistance() (int, bool) {
	stackTop, stackExists := c.Stack().Peek()
	queueTop, queueExists := c.Queue().Peek()
	if stackExists && queueExists {
		dist := queueTop - stackTop
		// "normalize" to
		// 0 1 2 3 4 5 ... 10 ...
		// 0 1 2 3 4 ---5--  --- 6 ---
		if dist < 0 {
			dist = -dist
		}
		switch {
		case dist > 10:
			return 6, true
		case dist > 5:
			return 5, true
		default:
			return dist, true
		}
	}
	return 0, false
}

func (c *SimpleConfiguration) GetSource(location byte) Index {
	switch location {
	case 'N':
		return c.Queue()
	case 'S':
		return c.Stack()
	}
	return nil
}

func (c *SimpleConfiguration) GetHead(nodeID int) (*ArcCachedDepNode, bool) {
	head := c.Nodes[nodeID].Head
	if head == -1 {
		return nil, false
	}
	return c.Nodes[head], true
}

func (c *SimpleConfiguration) GetModifiers(nodeID int) ([]int, []int) {
	node := c.Nodes[nodeID]
	return node.LeftMods(), node.RightMods()
}

func (c *SimpleConfiguration) GetNumModifiers(nodeID int) (int, int) {
	node := c.Nodes[nodeID]
	return len(node.LeftMods()), len(node.RightMods())
}

func (c *SimpleConfiguration) GetModifierLabelSets(nodeID int) (interface{}, interface{}, interface{}) {
	node := c.Nodes[nodeID]
	return node.LeftLabelSet(), node.RightLabelSet(), node.AllLabelSet()
}

func (c *SimpleConfiguration) GetModifiersPOS(nodeID int) interface{} {
	node := c.Nodes[nodeID]
	return node.AllModPOS()
}
