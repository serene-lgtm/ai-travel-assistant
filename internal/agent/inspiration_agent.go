package agent

import (
	"context"
	"fmt"
	"strings"

	"ai-reading-assistant/internal/llm"
	"ai-reading-assistant/internal/model"
)

type InspirationAgent interface {
	Generate(ctx context.Context, session *model.InspirationSession, ragContext *RAGContext) (string, error)
}

type inspirationAgent struct {
	llmClient chatCaller
}

func NewInspirationAgent(client *llm.DeepseekClient) InspirationAgent {
	return &inspirationAgent{llmClient: client}
}

func (a *inspirationAgent) Generate(_ context.Context, session *model.InspirationSession, ragContext *RAGContext) (string, error) {
	if a.llmClient == nil {
		return "", fmt.Errorf("inspiration agent llm client is not initialized")
	}
	prompt, err := BuildInspirationPrompt(session, ragContext)
	if err != nil {
		return "", err
	}
	resp, err := a.llmClient.Call(prompt)
	if err != nil {
		return "", fmt.Errorf("generate inspiration: %w", err)
	}
	return stripMarkdownToPlainText(resp), nil
}

func BuildInspirationPrompt(session *model.InspirationSession, ragContext *RAGContext) (string, error) {
	if session == nil {
		return "", fmt.Errorf("session is nil")
	}
	current, ok := session.CurrentRequirement()
	if !ok {
		return "", fmt.Errorf("current requirement is empty")
	}

	getContent := func(field model.RequirementField) string {
		item := current.Get(field)
		content := strings.TrimSpace(item.Content)
		if strings.TrimSpace(item.SelectedOption) != "" {
			if content == "" {
				content = item.SelectedOption
			} else {
				content += "\n" + strings.TrimSpace(item.SelectedOption)
			}
		}
		return content
	}

	prompt := fmt.Sprintf(genInspirationPrompt,
		getContent(model.RequirementFieldMood),
		getContent(model.RequirementFieldScene),
		getContent(model.RequirementFieldFocus),
		buildRAGPromptSection(ragContext),
	)
	return prompt, nil
}

func buildRAGPromptSection(ragContext *RAGContext) string {
	if ragContext == nil || strings.TrimSpace(ragContext.ReferenceText) == "" {
		return "无"
	}
	return ragContext.ReferenceText
}

const genInspirationPrompt = `
你的role: ` + RoleIteraryTripPlanner + `
请根据以下需求写出一段200-400字的旅行灵感描述,希望能具有严肃文学的气质,同时也不乏美感,让这样一段灵感具有令人心驰神往的力量。
可以根据不同的情感基调让你的描述具有不同的质感:
追溯历史的mood--纪录片的质感;
追寻诗歌和文学的足迹--充满诗意和想象力;
对自然的感受--震撼人心的,壮阔的;
户外探险的体验--有活力的,充满生命力的;
人文相关的体验--抚慰人心又不失深刻.
当然具体情况具体分析,每一种质感都请贴近用户的原始表述.
[情感基调]
%s
[旅行场景]
%s
[核心关注点/行动]
%s
[参考资料]
%s
要求:
1. 以中文输出,给这段文字起一个标题,并在第一段用1-3句话对内容进行简洁概括。
2. 不要使用 markdown,不要输出标题符号、加粗符号、列表符号、代码块或链接格式,只输出纯文本。
3. 如果目的景点有一定的历史文化背景,请先简单科普介绍。
4. 内容中要体现可执行的场景细节(季节/时间/地点/动作),其中地点一定要很精确,如县/小镇/村庄/城市的区域。
5. 当介绍景点时，用无人称的客观描述语气，不要用“你”“我”“他”等人称代词，直接描述这个地方本身。
6. 整篇只围绕一个核心地点或一个高度连贯的区域展开,不要并列推荐多个分散地点。
7. 如果核心关注点涉及作家、作品或文学人物,在已确定地点的前提下补充该地点与其相关的生活痕迹、创作背景或文学联想。
8. 如果用户在寻找作家生活痕迹、写作氛围或作品中的空间回声,请尽量具体说明: 哪位作家或作品与此地有关,曾在此发生过什么,当地哪些街区/旧居/书店/码头/路径仍能承接这种联系。
9. 文学关联要服务于旅行落地,不要空泛堆砌典故;优先写可抵达、可观察、可停留的空间细节。
10. 如果参考资料提供了地点背景或文学/历史关联,优先据此生成;如果参考资料不足或不确定,不要编造具体事实。
11. 参考资料只用于增强地点背景、空间细节和关联依据,不要把成文写成百科摘要。
`

const RoleIteraryTripPlanner = `
你是一位富有经验的旅游者，你对文学和艺术颇有审美与见地。
你践行着“读万卷书，行万里路”的宗旨，在行走中思考，在思考中构建着内心世界的秩序。
你对美有着执着的追求和敏锐的感知，心中的山河和眼中的山河共振的时刻，对你来说就是吉光片羽的圆满瞬间。
你想要把这一份感受传递给他人，你善于将别人模糊的关于旅行的渴望转变为具体的灵感。
你将竭尽全力帮助他们。
`
