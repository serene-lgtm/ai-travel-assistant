# Prompt Eval

`eval/` 目录用于维护 live prompt eval，不属于生产运行路径。

## 运行

全量运行：

```bash
PROMPT_EVAL=1 go test ./eval -run TestPromptEvalRun -v
```

只跑指定 case：

```bash
PROMPT_EVAL=1 PROMPT_EVAL_CASES=intent_001,clarify_004 go test ./eval -run TestPromptEvalRun -v
```

比较两个结果文件：

```bash
PROMPT_EVAL_COMPARE=1 \
PROMPT_EVAL_BASELINE=artifacts/prompt_eval/a.json \
PROMPT_EVAL_CANDIDATE=artifacts/prompt_eval/b.json \
go test ./eval -run TestPromptEvalCompare -v
```

## 目录

- `eval/testdata/prompt_eval/*.json`：评测样本
- `artifacts/prompt_eval/*.json`：运行结果

## 人工评分

每个结果文件都允许回填：

- `manual_score`
- `manual_notes`

### intent

- `intent_correct`: `pass|fail`

### clarification

- `target_field_correct`: `pass|fail`
- `question_quality`: `1-5`
- `options_quality`: `1-5`
- `scene_option_purity`: `1-5`

### inspiration

- `intent_fit`: `1-5`
- `single_place_grounding`: `1-5`
- `actionable_detail`: `1-5`
- `literary_quality`: `1-5`
- `hallucination_check`: `pass|fail`
