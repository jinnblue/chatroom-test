package popular

import (
	"time"
)

// freqWord 高频词节点
type freqWord struct {
	word    string // 单词
	enterAt int64  // 节点加入时间(纳秒)
}

// MostPopularWord 高频词管理器
type MostPopularWord struct {
	needCalc     bool           // 需重新计算
	maxFreq      int            // 最高频单词频率
	lastCheckAt  int64          // 上次检查时间(纳秒)
	holdDuration time.Duration  // 最近对应截取的时间长度
	topWord      string         // 最高频单词
	wordsQueue   []*freqWord    // 最近高频词节点队列
	wordsFreqMap map[string]int // 最近高频词出现频率
}

// NewMostPopularWord 根据截取时长初始化高频词管理器
func NewMostPopularWord(hold time.Duration) *MostPopularWord {
	return &MostPopularWord{
		needCalc:     false,
		maxFreq:      0,
		lastCheckAt:  time.Now().UnixNano(),
		holdDuration: hold,
		topWord:      "",
		wordsQueue:   make([]*freqWord, 0, 1024),
		wordsFreqMap: make(map[string]int, 1024),
	}
}

// Record 记录单词,若距离上次检查超过1秒进行截取处理
func (m *MostPopularWord) Record(word string) {
	now := time.Now().UnixNano()
	fw := &freqWord{
		word:    word,
		enterAt: now,
	}
	m.wordsQueue = append(m.wordsQueue, fw)
	m.wordsFreqMap[word]++
	freq := m.wordsFreqMap[word]
	if freq > m.maxFreq {
		m.maxFreq = freq
		m.topWord = word
		m.needCalc = false
	}

	if time.Duration(now-m.lastCheckAt) > time.Second {
		past := now - m.holdDuration.Nanoseconds()
		m.checkPos(past)
		// pos := m.checkPos(past)
		// log.Println("=======Record=>checkPos:", pos)
		m.lastCheckAt = now
	}
}

// GetTopWord 获取最近的最高频单词,按对应的最近时长截取
func (m *MostPopularWord) GetTopWord(lately time.Duration) string {
	if lately > m.holdDuration {
		lately = m.holdDuration
	}
	past := time.Now().Add(-lately).UnixNano()
	m.checkPos(past)
	// pos := m.checkPos(past)
	// log.Println("=======GetTopWord=>checkPos:", pos)
	if !m.needCalc {
		return m.topWord
	}

	max := 0
	most := ""
	for word, freq := range m.wordsFreqMap {
		if freq > max {
			most = word
			max = freq
		}
	}
	m.maxFreq = max
	m.topWord = most
	m.needCalc = false
	return most
}

// checkPos 根据时间获得截取位置
func (m *MostPopularWord) checkPos(past int64) int {
	pos := 0
	for i := 0; i < len(m.wordsQueue); i++ {
		fw := m.wordsQueue[i]
		if fw.enterAt >= past {
			break
		}
		freq := m.wordsFreqMap[fw.word]
		if freq >= m.maxFreq {
			m.needCalc = true
		}
		freq--
		if freq <= 0 {
			delete(m.wordsFreqMap, fw.word)
		} else {
			m.wordsFreqMap[fw.word] = freq
		}
		pos++
	}
	if pos > 0 {
		m.wordsQueue = m.wordsQueue[pos:]
	}
	return pos
}
