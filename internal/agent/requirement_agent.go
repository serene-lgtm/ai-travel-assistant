package agent

import (
	"context"
	"fmt"
	"strings"

	"ai-reading-assistant/internal/llm"
	"ai-reading-assistant/internal/model"
)

type RequirementAnalyzerAgent interface {
	Analyze(ctx context.Context, msg *model.InspirationMessage, session *model.InspirationSession, targetField model.RequirementField) error
}

type requirementAnalyzerAgent struct {
	llmClient chatCaller
}

func NewRequirementAnalyzerAgent(client *llm.DeepseekClient) RequirementAnalyzerAgent {
	return &requirementAnalyzerAgent{llmClient: client}
}

func (a *requirementAnalyzerAgent) Analyze(_ context.Context, msg *model.InspirationMessage, session *model.InspirationSession, targetField model.RequirementField) error {
	if msg == nil {
		return fmt.Errorf("message is nil")
	}
	if a.llmClient == nil {
		return fmt.Errorf("requirement agent llm client is not initialized")
	}
	if session == nil {
		return fmt.Errorf("session is nil")
	}
	current := session.EnsureCurrentRequirement()
	if current == nil {
		return fmt.Errorf("current requirement is nil")
	}

	clarifying := targetField != ""

	if clarifying && msg.Kind == model.MessageKindUserChoice && len(msg.Options) > 0 {
		return a.applyUserChoice(current, targetField, msg)
	}

	return a.analyzeFreeTextInput(current, targetField, msg)
}

func (a *requirementAnalyzerAgent) applyUserChoice(current *model.Inspiration, targetField model.RequirementField, msg *model.InspirationMessage) error {
	selected := strings.TrimSpace(msg.Options[0].Content)
	if selected == "" {
		return fmt.Errorf("selected option content is empty")
	}

	// targetField points to the requirement slot the current choice should fill.
	item := current.Get(targetField)
	item.SelectedOption = selected
	if item.Score < requirementSatisfiedScore {
		item.Score = requirementSatisfiedScore
	}
	current.Set(targetField, item)
	return nil
}

func (a *requirementAnalyzerAgent) analyzeFreeTextInput(current *model.Inspiration, targetField model.RequirementField, msg *model.InspirationMessage) error {
	inputText := strings.TrimSpace(msg.Content)
	if inputText == "" {
		return fmt.Errorf("content is empty")
	}

	flows := []struct {
		field            model.RequirementField
		extractionPrompt string
		scoringPrompt    string
	}{
		{field: model.RequirementFieldFocus, extractionPrompt: focusExtractionPrompt, scoringPrompt: focusScoringPrompt},
		{field: model.RequirementFieldMood, extractionPrompt: moodExtractionPrompt, scoringPrompt: moodScoringPrompt},
		{field: model.RequirementFieldScene, extractionPrompt: sceneExtractionPrompt, scoringPrompt: sceneScoringPrompt},
	}

	for _, flow := range flows {
		// When targetField is set, only this field needs to be clarified.
		if targetField != "" && flow.field != targetField {
			continue
		}

		value, score, err := a.extractRequirementField(flow.field, flow.extractionPrompt, flow.scoringPrompt, inputText)
		if err != nil {
			return fmt.Errorf("extract %s: %w", flow.field, err)
		}

		item := current.Get(flow.field)
		item.Content = value
		item.Score = score
		current.Set(flow.field, item)
	}

	if targetField != "" && current.Get(targetField).Content == "" {
		return fmt.Errorf("clarify %s produced empty content", targetField)
	}
	return nil
}

func (a *requirementAnalyzerAgent) extractRequirementField(field model.RequirementField, extractionTemplate, scoringGuide, userInput string) (string, int, error) {
	extractionPrompt := fmt.Sprintf(extractionTemplate, userInput)
	prompt := fmt.Sprintf(`%s

请忽略上文原有的输出格式要求,统一只输出 JSON {"content":"...", "score": <0-5 的整数>}.
说明:
- content 是你依据上述指引为[%s]提炼出的关键信息,若无法提取请设为""。
- score 按以下标准给出0-5之间的整数,0表示信息缺失:
%s
`, extractionPrompt, fieldLabels[field], scoringGuide)

	raw, err := a.llmClient.Call(prompt)
	if err != nil {
		return "", 0, err
	}

	var payload map[string]any
	if err := decodeFirstJSONObject(raw, &payload); err != nil {
		return "", 0, err
	}

	content := ""
	if val, ok := payload["content"].(string); ok {
		content = strings.TrimSpace(val)
	}
	return content, normalizeScore(payload["score"]), nil
}

var fieldLabels = map[model.RequirementField]string{
	model.RequirementFieldMood:  "情感基调",
	model.RequirementFieldScene: "旅行场景",
	model.RequirementFieldFocus: "核心焦点",
}

const extractionPromptRole = `
你是一个资深的文学旅行策划师,具有深厚的文学/艺术/哲学素养.你的任务是分析用户的旅行请求,从文学艺术自然等相关角度深入理解其意图.
核心能力:
**地理文学知识库**:掌握作家,作品与地点的关联
   - 村上春树:东京,希腊小岛,挪威森林
   - 唐诗:长安(西安),洛阳,扬州,襄阳,边塞(河西走廊)
   - 三毛:撒哈拉(摩洛哥西撒哈拉),加纳利群岛,中南美
请仅做高置信度的推导.
高置信度:"我想去追寻一下书本里汪曾祺故乡的滋味",可以推出地点是江苏高邮,这是可以的;
低置信度:"我想去诗意的地方进行一场文学之旅",推荐瓦尔登湖,阿尔勒,这是禁止的.
`

const (
	focusExtractionPrompt = extractionPromptRole + `
请深入理解用户的意图,提取旅行的核心焦点.
1.用户期望在旅行中进行的核心活动:
    - 感官体验:看什么,听什么,闻什么,感受什么
    - 具体动作:徒步,禅修,书写,拍摄,寻找,等待
    - 时间要素:季节,月份,具体时刻,时长
    - 仪式性行为:告别,纪念,庆祝,反思
2.用户提及的具体作品/人物/细节:
    - 作品示例:画作,小说,诗歌,电影,文章,歌曲,纪录片等
    - 人物示例:艺术家,作家,导演,电影或书籍角色,演员
    - 细节示例:梵高的"星空"在阿尔勒时期的创作
	- 意象示例:感受诗意的生活,探寻艺术家的童年

    要求:
    1. 只返回 JSON:{"focus": "<作品或人物,多个用,分隔;若无则空字符串>"}
    2. 只列最关键的1-3个指涉.

    用户请求:
    """%s"""`

	moodExtractionPrompt = extractionPromptRole + `
请深入感受,理解并判断/推断用户描述所偏好的情感基调或是氛围,可以归纳为以下的例子或其他:
    - 文学/艺术
    - 历史/人文/哲思
    - 诗意/浪漫
    - 磅礴/史诗/传奇
    - 温暖/治愈/慰藉人心/积极
    - 活泼/明快/鲜艳
    - 清冷/孤独/沉静
    - 怀旧/感伤
    - 震撼/壮阔
    - 探险/刺激
    - 其他具体描述

    要求:
    1. 只返回 JSON:{"mood": "<情感基调,若无则空字符串>"}
    2. 以上是一些归纳,但用户原本的表述可能更丰满,请保留用户原本更细化的表述,如果没有,mood设为"".

    用户请求:
    """%s"""`

	sceneExtractionPrompt = extractionPromptRole + `
请深入分析用户的旅行意图,提取出用户描述的旅行地点,请给出你能判断出的最精确范围.例如:中国乡镇请不要返回中国,而是返回中国乡镇.
 1. 用户明确提及或可高置信度推导的具体地点,如国家,地区,具体景点
 2. 描述或暗示的环境氛围特征,如雪中小镇,古老街区.
 3. 对环境和地貌的描述,如海边悬崖,森林深处,峡湾地貌.
    请不要做置信度低的推导.

要求:
1. 只返回 JSON:{"scene": "<地点,多个用,分隔;若无则空字符串>"}
2. 请返回最相关的1-3个地点,如果没有提及地点或无法确定,scene设为"".

用户请求:
"""%s"""`
)

const (
	focusScoringPrompt = `
你是一个"核心焦点"评分专家.请评估焦点描述的明确度.
5分[完美具体]: 文学艺术锚点 + 具体动作 + 感官细节 + 时间/地点特征
4分[高度具体]: 具体作品/人物 + 具体动作, 或具体动作 + 独特感官/情感描述, 或具体时空 + 具体体验意向
3分[基本具体]: 具体作品/人物, 或具体动作 + 基本感官, 或具体时间/地点偏好
2分[模糊]: 基本动作或模糊意向
1分[极模糊]: 几乎无信息
0分: 完全空字符串
`

	moodScoringPrompt = `
你是一个"情感基调"评分专家.请评估情感描述的明确度.
5分[深刻具体]: 具体情感 + 文学艺术参照 + 个人化表达
4分[具体鲜明]: 具体情感 + 质地描述 或 明确的体验类型
3分[基本明确]: 基本情感方向或体验类型
2分[模糊]: 模糊的情感/氛围词
1分[极模糊]: 几乎无信息
0分: 完全空字符串
`

	sceneScoringPrompt = `
你是一个"旅行场景"评分专家.请评估地点描述的明确度.
5分[精确具体]: 具体坐标/景点 + 环境特征 + 时间窗口
4分[高度具体]: 具体城市/景点 + 环境特征, 或具体城市/区域 + 时间特征
3分[基本具体]: 具体城市/区域, 或具体国家 + 区域/地貌特征
2分[模糊]: 大范围地区、模糊环境、广泛地貌特征
1分[极模糊]: 几乎无信息
0分: 完全空字符串
`
)
