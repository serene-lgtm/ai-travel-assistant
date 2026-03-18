package agent

import (
	"context"
	"fmt"
	"strings"

	"ai-reading-assistant/internal/llm"
	"ai-reading-assistant/internal/model"
)

type InspirationAgent interface {
	Generate(ctx context.Context, session *model.InspirationSession) (string, error)
}

type inspirationAgent struct {
	llmClient chatCaller
}

func NewInspirationAgent(client *llm.DeepseekClient) InspirationAgent {
	return &inspirationAgent{llmClient: client}
}

func (a *inspirationAgent) Generate(_ context.Context, session *model.InspirationSession) (string, error) {
	if a.llmClient == nil {
		return "", fmt.Errorf("inspiration agent llm client is not initialized")
	}
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
	)

	resp, err := a.llmClient.Call(prompt)
	if err != nil {
		return "", fmt.Errorf("generate inspiration: %w", err)
	}
	return stripMarkdownToPlainText(resp), nil
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
要求:
1. 以中文输出,给这段文字起一个标题,并在第一段用1-3句话对内容进行简洁概括。
2. 不要使用 markdown,不要输出标题符号、加粗符号、列表符号、代码块或链接格式,只输出纯文本。
3. 如果目的景点有一定的历史文化背景,请先简单科普介绍。
4. 内容中要体现可执行的场景细节(季节/时间/地点/动作),其中地点一定要很精确,如县/小镇/村庄/城市的区域。
5. 请使用第三人称的视角进行介绍。
`

const RoleIteraryTripPlanner = `
你是一位富有经验的旅游者，你对文学和艺术颇有审美与见地。
你践行着“读万卷书，行万里路”的宗旨，在行走中思考，在思考中构建着内心世界的秩序。
你对美有着执着的追求和敏锐的感知，心中的山河和眼中的山河共振的时刻，对你来说就是吉光片羽的圆满瞬间。
你想要把这一份感受传递给他人，你善于将别人模糊的关于旅行的渴望转变为具体的灵感。
你将竭尽全力帮助他们。
`
