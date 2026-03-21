package analytics

import (
	"math"
	"regexp"
	"sort"
	"strings"
	"unicode"

	"github.com/go-ego/gse"
)

// KeywordExtractor 关键词提取器
type KeywordExtractor struct {
	seg         gse.Segmenter
	filterWords map[string]bool
}

// KeywordFreq 关键词频率
type KeywordFreq struct {
	Word  string
	Count int
}

// NewKeywordExtractor 创建关键词提取器
func NewKeywordExtractor() *KeywordExtractor {
	var seg gse.Segmenter
	seg.LoadDict()
	return &KeywordExtractor{
		seg:         seg,
		filterWords: buildFilterWords(),
	}
}

// Close 关闭分词器
func (ke *KeywordExtractor) Close() {}

// ExtractTopKeywords 提取 Top N 关键词（TF-IDF）
func (ke *KeywordExtractor) ExtractTopKeywords(texts []string, topN int) []KeywordFreq {
	if len(texts) == 0 {
		return []KeywordFreq{}
	}

	totalWordCount := make(map[string]int)
	docCount := make(map[string]int)

	for _, text := range texts {
		words := ke.seg.Cut(text, true)
		seen := make(map[string]bool)
		for _, word := range words {
			word = strings.TrimSpace(word)
			if !ke.isValid(word) {
				continue
			}
			totalWordCount[word]++
			if !seen[word] {
				docCount[word]++
				seen[word] = true
			}
		}
	}

	numDocs := float64(len(texts))
	minDocFreq := 2
	if len(texts) <= 10 {
		minDocFreq = 1
	}

	scores := make(map[string]float64)
	for word, tf := range totalWordCount {
		if docCount[word] < minDocFreq {
			continue
		}
		idf := math.Log(numDocs/float64(docCount[word])) + 1
		scores[word] = float64(tf) * idf
	}

	keywords := make([]KeywordFreq, 0, len(scores))
	for word, score := range scores {
		keywords = append(keywords, KeywordFreq{Word: word, Count: int(score * 100)})
	}
	sort.Slice(keywords, func(i, j int) bool {
		return keywords[i].Count > keywords[j].Count
	})
	if len(keywords) > topN {
		keywords = keywords[:topN]
	}
	return keywords
}

var mixedNumChinese = regexp.MustCompile(`^[\d\p{Han}]+$`)

// isValid 综合过滤
func (ke *KeywordExtractor) isValid(word string) bool {
	runes := []rune(word)
	if len(runes) < 2 || len(runes) > 8 {
		return false
	}
	if ke.filterWords[word] {
		return false
	}
	if isNumber(word) || isPunctuation(word) || isPureEnglish(word) {
		return false
	}
	if containsSpecialChars(word) {
		return false
	}
	if !containsChinese(word) {
		return false
	}
	// 过滤纯数字+汉字混合但无实意的词（如"第3""图1"）
	hasDigit := false
	for _, r := range runes {
		if unicode.IsDigit(r) {
			hasDigit = true
			break
		}
	}
	if hasDigit && len(runes) <= 3 {
		return false
	}
	return true
}

// buildFilterWords 构建综合过滤词表
func buildFilterWords() map[string]bool {
	words := []string{
		// ===== 微信/新媒体文章元词 =====
		"链接", "点击", "阅读", "原文", "全文", "来源", "声明", "版权", "转载",
		"本文", "本期", "本号", "本报", "本刊", "本网", "本站",
		"小编", "编辑", "作者", "记者", "责编", "主编", "图片",
		"关注", "订阅", "分享", "转发", "点赞", "在看", "收藏",
		"回复", "留言", "评论", "投稿", "联系", "爆料",
		"长按", "扫码", "二维码", "识别", "扫描",
		"往期", "精选", "合集", "汇总", "上期", "下期", "上篇", "下篇",
		"推荐", "延伸", "相关阅读", "更多", "精彩",
		"免责", "侵权", "授权", "联系我们",

		// ===== 新闻套话 =====
		"据悉", "据了解", "据报道", "据称", "消息",
		"日前", "近日", "近期", "此前", "早前", "最新", "最近",
		"举行", "召开", "出席", "主持", "参加",
		"针对", "围绕", "就此", "对此", "为此",

		// ===== 基础虚词 =====
		"的", "了", "在", "是", "我", "有", "和", "就", "不", "人", "都",
		"上", "也", "很", "到", "说", "要", "去", "你", "会", "着", "没有",
		"自己", "这", "那", "里", "为", "以", "用", "来", "时", "地", "可以",
		"这个", "中", "出", "而", "能", "对", "多", "然后", "她", "他", "但是",

		// ===== 连词介词 =====
		"与", "及", "等", "被", "从", "由", "于", "将", "或", "把", "让", "给",
		"如", "若", "则", "且", "又", "之", "所", "其", "某", "该", "每", "各",
		"以及", "并且", "虽然", "因为", "所以", "如果", "只要", "既然",
		"因此", "从而", "然而", "不过", "而且", "即使", "尽管",
		"此外", "另外", "以上", "以下", "综上", "总之",
		"关于", "对于", "根据", "按照", "通过", "经过", "为了",
		"首先", "其次", "最后", "总的来说",

		// ===== 时间副词 =====
		"已经", "正在", "将要", "曾经", "刚刚", "仍然",
		"现在", "过去", "将来", "以前", "以后", "之前", "之后", "当时",
		"今天", "昨天", "明天", "今年", "去年", "明年",

		// ===== 程度副词 =====
		"非常", "十分", "特别", "尤其", "格外", "更加", "越来越", "比较", "相当",
		"稍微", "略微", "有点", "一些", "极其", "高度",
		"大约", "超过", "不足", "将近",

		// ===== 助词语气词 =====
		"啊", "呀", "吗", "吧", "呢", "哦", "哈", "嘛", "啦", "嗯",

		// ===== 代词 =====
		"我们", "你们", "他们", "她们", "咱们", "大家", "别人", "其他",
		"本人", "彼此", "相互", "这些", "那些", "这样", "那样",
		"这里", "那里",

		// ===== 泛化动词 =====
		"提取", "获取", "获得", "取得", "处理", "分析", "实现", "完成",
		"进行", "操作", "执行", "运行", "启动", "触发", "调用", "使用",
		"建立", "构建", "创建", "生成", "产生", "形成", "造成",
		"导致", "引起", "带来", "引发", "改变", "转变",
		"提高", "提升", "增加", "增强", "加强", "扩大", "拓展",
		"降低", "减少", "消除", "解决",
		"推进", "推动", "促进", "加快", "推广", "传播",
		"开展", "开始", "继续", "坚持", "保持", "维护", "保护",
		"支持", "帮助", "协助", "配合", "参与", "参加",
		"发布", "发表", "公布", "公开", "披露", "宣布", "宣传",
		"表示", "表明", "强调", "指出", "提出", "提到", "说明",
		"呼吁", "要求", "建议", "倡议", "号召", "鼓励",
		"了解", "掌握", "学习", "探索", "调查", "检查",
		"确保", "保证", "防止", "避免", "应对",
		"发现", "显示", "表现", "体现", "反映", "证明",
		"认为", "觉得", "感觉", "以为", "知道", "看到", "听到",
		"希望", "打算", "准备", "计划", "预计", "期待",
		"停止", "结束", "终止", "关闭", "中断", "暂停",
		"做到", "搞好", "成为", "变成", "得到", "达到",
		"采用", "采取", "运用", "利用", "借助",

		// ===== 形容词（过于宽泛）=====
		"新", "旧", "大", "小", "多", "少", "高", "低", "长", "短", "好", "坏",
		"相关", "相应", "不同", "各种", "多种",

		// ===== 量词 =====
		"位", "名", "条", "项", "件", "次", "遍", "回", "场",
		"种", "类", "样", "份", "批", "群", "队", "组",

		// ===== 空洞名词 =====
		"信息", "数据", "材料", "资料", "系统", "平台", "渠道",
		"手段", "措施", "办法", "方案", "方式", "方法", "过程", "结果",
		"原因", "目的", "作用", "意义", "价值", "内容", "形式",
		"特点", "特征", "范围", "领域", "方面", "情况", "问题", "工作",
		"事情", "地方", "时候", "状态", "形势", "关系", "联系",
		"程度", "水平", "层次", "等级", "部分", "全部", "整体",
		"条件", "环境", "背景", "基础", "前提", "保障",
		"时间", "日期", "年份", "月份", "位置", "地点", "场所", "区域",
		"数量", "数字", "总数", "事物",
	}

	m := make(map[string]bool, len(words))
	for _, w := range words {
		m[w] = true
	}
	return m
}

func isNumber(s string) bool {
	if len(s) == 0 {
		return false
	}
	for _, r := range s {
		if !unicode.IsDigit(r) {
			return false
		}
	}
	return true
}

func isPunctuation(s string) bool {
	if len(s) == 0 {
		return false
	}
	for _, r := range s {
		if !unicode.IsPunct(r) && !unicode.IsSymbol(r) {
			return false
		}
	}
	return true
}

func isPureEnglish(s string) bool {
	if len(s) == 0 {
		return false
	}
	for _, r := range s {
		if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z')) {
			return false
		}
	}
	return true
}

func containsChinese(s string) bool {
	for _, r := range s {
		if unicode.Is(unicode.Han, r) {
			return true
		}
	}
	return false
}

func containsSpecialChars(s string) bool {
	pattern := regexp.MustCompile(`[^\p{Han}\p{L}\p{N}]`)
	return pattern.MatchString(s)
}
