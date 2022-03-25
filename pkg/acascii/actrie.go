// acascii 包实现了 Aho-Corasick 字符串匹配算法,仅支持 ASCII;
// 默认忽略大小写并跳过非英文和数字字符;
// 代码中 []byte 被称为一个 Words(字典/敏感词);
package acascii

import (
	"bufio"
	"bytes"
	"errors"
	"io"
	"os"
	"unsafe"
)

type Words []byte

// ErrNotASCII 当输入的 Words 不是 ASCII 时返回
var ErrNotASCII = errors.New("not-ASCII input")

// DEFAULT_SKIP_SYMBOL 默认跳过非英文和数字字符
const DEFAULT_SKIP_SYMBOL = true

type emitState byte

const (
	IS_EMIT emitState = 1 << iota
	HAS_EMIT
)
const ALL_EMIT = IS_EMIT | HAS_EMIT

const MAX_CHAR = 128

// acState 用于实现 AC自动机 trie 结构中的一个节点
type acState struct {
	emit    emitState          // isEmit 表示匹配成功; hasEmit 表示包含某个匹配
	depth   uint16             // 0 表示根节点; >0 表示词长
	failure *acState           // 指向字典后缀可匹配的下一个节点指针,用于匹配失败时在trie中回退
	success [MAX_CHAR]*acState // 数组中非空元素的索引表示追加的字节值,trie中的Words由这些元素(子节点指针)逐字节构建而成
}

// ACTrie AC自动机,包含要匹配的 Words 链表
type ACTrie struct {
	rootState  *acState // 指向根节点
	inited     bool     // true 表示构建完成
	skipSymbol bool     // true 表示跳过非英文和数字字符
}

// newACTrie 创建根节点并初始化AC自动机
func newACTrie() *ACTrie {
	root := new(acState)
	root.failure = root

	return &ACTrie{
		rootState:  root,
		inited:     false,
		skipSymbol: DEFAULT_SKIP_SYMBOL,
	}
}

// NewACTrieFromFile 从文件中加载字典并完成AC自动机的构建
// 字典包含非ASCII字符时 panic
func NewACTrieFromFile(filename string) *ACTrie {
	f, err := os.Open(filename)
	if err != nil {
		// return nil
		panic(err)
	}

	ac := newACTrie()
	r := bufio.NewReader(f)
	for {
		word, err := r.ReadBytes('\n')
		if err != nil || err == io.EOF {
			break
		}
		word = bytes.TrimSpace(word)
		acErr := ac.AddKeyWord(word)
		if acErr != nil {
			panic(acErr)
		}
	}

	if ac.checkFailuresCreated() {
		return ac
	}
	return nil
}

// NewACTrieFromStrings 通过字典切片完成AC自动机的构建
// 字典包含非ASCII字符时返回 ErrNotASCII
func NewACTrieFromStrings(dict []string) (*ACTrie, error) {
	ac := newACTrie()
	for _, word := range dict {
		err := ac.AddKeyWord(Words(word))
		if err != nil {
			return nil, err
		}
	}
	if ac.checkFailuresCreated() {
		return ac, nil
	} else {
		return nil, errors.New("ACTrie init fail")
	}
}

// isSymbolASCII 判断非英文和数字字符
func isSymbolASCII(char byte) bool {
	if (('0' <= char) && (char <= '9')) ||
		(('a' <= char) && (char <= 'z')) ||
		(('A' <= char) && (char <= 'Z')) {
		return false
	}
	return true
}

// toLowerASCII 将英文转为小写
func toLowerASCII(char byte) byte {
	if ('A' <= char) && (char <= 'Z') {
		char += 'a' - 'A'
	}
	return char
}

// AddKeyWord 追加Words,将会重新构建AC自动机
func (ac *ACTrie) AddKeyWord(word Words) error {
	if len(word) <= 0 {
		return nil
	}

	curr := ac.rootState
	for _, char := range word {
		if char >= MAX_CHAR {
			return ErrNotASCII
		}
		if ac.skipSymbol && isSymbolASCII(char) {
			continue
		}
		curr = curr.addLeaf(char)
	}
	// 在字典词尾添加命中标志
	if curr.depth > 0 {
		curr.setIsEmit()
	}
	ac.inited = false
	return nil
}

func Bytes2String(b []byte) string {
	return *(*string)(unsafe.Pointer(&b))
}

// Filter 将字符串中匹配到的Words替换为 "*"
func (ac *ACTrie) Filter(input string) string {
	if !ac.checkFailuresCreated() {
		return input
	}

	n := 0
	symbolPre := false
	curr := ac.rootState
	word := Words(input)
	for pos := 0; pos < len(word); pos++ {
		n++
		char := word[pos]
		if ac.skipSymbol && isSymbolASCII(char) {
			symbolPre = true
			continue
		}
		for {
			//先从success跳转
			next := ac.nextState(curr, char)
			if next != nil {
				curr = next
				break
			}
			//跳转失败,从failure跳转
			curr = curr.failure
			if (!symbolPre) && curr.isEmit() {
				pos--
				n = int(curr.depth) - 1
				break
			}
		}
		if curr.isWordsHead() {
			n = 0
		}
		if curr.isEmit() {
			for j := pos - n; j <= pos; j++ {
				word[j] = '*'
			}
			n = 0
		}
		symbolPre = false
	}

	// return string(word)
	return Bytes2String(word)
}

// HasBlackWord 判断字符串中是否包含Words
func (ac *ACTrie) HasBlackWord(input string) bool {
	if !ac.checkFailuresCreated() {
		return false
	}

	word := Words(input)
	curr := ac.rootState
	for _, char := range word {
		if ac.skipSymbol && isSymbolASCII(char) {
			continue
		}
		for {
			//先从success跳转
			next := ac.nextState(curr, char)
			if next != nil {
				curr = next
				break
			}
			//跳转失败,从failure跳转
			curr = curr.failure
		}
		if curr.hasEmit() {
			return true
		}
	}
	return false
}

// BytesHasBlackWord 判断字符串中是否包含Words
// 直接传入字符串转换后的字节数组切片,用于性能测试
func (ac *ACTrie) BytesHasBlackWord(word Words) bool {
	if !ac.checkFailuresCreated() {
		return false
	}

	curr := ac.rootState
	for _, char := range word {
		if ac.skipSymbol && isSymbolASCII(char) {
			continue
		}
		for {
			//先从success跳转
			next := ac.nextState(curr, char)
			if next != nil {
				curr = next
				break
			}
			//跳转失败,从failure跳转
			curr = curr.failure
		}
		if curr.hasEmit() {
			return true
		}
	}
	return false
}

// nextState 节点跳转
func (ac *ACTrie) nextState(curr *acState, char byte) *acState {
	var next *acState
	if char < MAX_CHAR {
		char = toLowerASCII(char)
		next = curr.success[char]
	}
	if (next == nil) && (curr.depth == 0) {
		return curr
	}
	return next
}

// checkFailuresCreated 检查并完成构建
func (ac *ACTrie) checkFailuresCreated() bool {
	if !ac.inited {
		ac.createFailureStates()
	}
	return ac.inited
}

// createFailureStates 构建匹配失败时的回退链
func (ac *ACTrie) createFailureStates() {
	queue := make([]*acState, 0, len(ac.rootState.success))
	//1. 深度=1的节点,failure指向rootState
	for _, sn := range ac.rootState.success {
		if sn != nil {
			depth1 := sn
			depth1.failure = ac.rootState
			queue = append(queue, depth1) //push
		}
	}

	//2. 为深度>1的节点建立failure表(BFS)
	for len(queue) > 0 {
		var curr *acState
		curr, queue = queue[0], queue[1:] //pop front
		//转向叶节点状态的char集合
		for k, state := range curr.success {
			if state == nil {
				continue
			}
			key := byte(k)
			next := state
			queue = append(queue, next)

			//由下而上找到S_Fail
			preFail := curr.failure
			for ac.nextState(preFail, key) == nil {
				preFail = preFail.failure
			}
			nextFail := ac.nextState(preFail, key)
			next.failure = nextFail
			//将包含词加入命中表,如ushe查[she,he]时,能一次匹配到所有词组
			if nextFail.hasEmit() {
				next.setHasEmit()
			}
		}
	}

	ac.inited = true
}

func (s *acState) setIsEmit() {
	s.emit = s.emit | IS_EMIT
}

func (s *acState) isEmit() bool {
	return (s.emit&IS_EMIT != 0)
}

func (s *acState) setHasEmit() {
	s.emit = s.emit | HAS_EMIT
}

func (s *acState) hasEmit() bool {
	return (s.emit&ALL_EMIT != 0)
}

// isWordsHead 判断节点是否字典起始字符
func (s *acState) isWordsHead() bool {
	return s.depth == 1
}

// addLeaf 追加子节点
func (s *acState) addLeaf(char byte) *acState {
	char = toLowerASCII(char)
	state := s.success[char]
	if state == nil {
		state = new(acState)
		state.depth = s.depth + 1
		s.success[char] = state
	}
	return state
}
