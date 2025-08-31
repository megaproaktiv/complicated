package reasoning

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime/types"
	"github.com/spf13/viper"
)

var Client *bedrockruntime.Client

func init() {
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		panic("configuration error, " + err.Error())
	}
	Client = bedrockruntime.NewFromConfig(cfg)
}

type Conversation struct {
	Question string `mapstructure:"question"`
	Answer   string `mapstructure:"answer"`
}

// Use the short answers and questions in config.yml
func RunShort() {
	// Use mapstructure with viper (https://github.com/mitchellh/mapstructure)
	var conversations []Conversation
	viper.UnmarshalKey("conversations", &conversations)

	ctx := context.Background()
	for i, c := range conversations {
		fmt.Printf("Question %v: %s\nAnswer %v: %s\n", i, c.Question, i, c.Answer)
		result := ApplyGuardrail(ctx, Client, c.Question, c.Answer, i)
		logFileName := path.Join("results", fmt.Sprintf("conversation_%d.log", i+1))
		os.WriteFile(logFileName, []byte(result), 0666)

	}
}

// use longer files in "chat"
func RunLong() {
	files, err := filepath.Glob("chat/*-question.txt")
	if err != nil {
		panic("error reading chat directory: " + err.Error())
	}

	var conversations []Conversation
	counter := 0
	for _, questionFile := range files {
		base := strings.TrimSuffix(questionFile, "-question.txt")
		answerFile := base + "-answer.txt"

		questionBytes, err := os.ReadFile(questionFile)
		if err != nil {
			panic("error reading question file: " + err.Error())
		}

		answerBytes, err := os.ReadFile(answerFile)
		if err != nil {
			panic("error reading answer file: " + err.Error())
		}

		conversations = append(conversations, Conversation{
			Question: string(questionBytes),
			Answer:   string(answerBytes),
		})
		counter++
	}
	fmt.Printf("Found %v chats\n", counter)

	ctx := context.Background()
	for i, c := range conversations {
		fmt.Printf("Question %v: %s\nAnswer %v: %s\n", i, c.Question, i, c.Answer)
		result := ApplyGuardrail(ctx, Client, c.Question, c.Answer, i)
		logFileName := path.Join("results", fmt.Sprintf("conversation_%d.log", i+1))
		os.WriteFile(logFileName, []byte(result), 0666)
	}
}
func ApplyGuardrail(ctx context.Context, client *bedrockruntime.Client, userQuery string, llmAnswer string, run int) string {
	// Implementation of Run function
	// Array index (0) vs human counting for print (1)
	run = run + 1
	id := viper.GetString("guradrail.id")
	version := viper.GetString("guradrail.version")
	parms := bedrockruntime.ApplyGuardrailInput{
		Content: []types.GuardrailContentBlock{
			&types.GuardrailContentBlockMemberText{
				Value: types.GuardrailTextBlock{
					Text: &userQuery,
					Qualifiers: []types.GuardrailContentQualifier{
						types.GuardrailContentQualifierQuery,
					},
				},
			},
			&types.GuardrailContentBlockMemberText{
				Value: types.GuardrailTextBlock{
					Text: &llmAnswer,
					Qualifiers: []types.GuardrailContentQualifier{
						types.GuardrailContentQualifierGuardContent,
					},
				},
			},
		},
		GuardrailIdentifier: &id,
		GuardrailVersion:    &version,
		// Checking the INPUT of the user query
		// or the OUTPUT of the LLM
		Source: types.GuardrailContentSourceOutput,
	}
	result, err := client.ApplyGuardrail(ctx, &parms)
	if err != nil {
		panic("error applying guardrail, " + err.Error())
	}
	// Print result in json format
	resultJson, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		panic("error marshaling result, " + err.Error())
	}

	findings := result.Assessments[0].AutomatedReasoningPolicy.Findings
	if len(findings) == 0 {
		fmt.Printf("OK - no findings\n")
	}
	for _, finding := range findings {
		fmt.Printf("%d: ", run)
		switch v := finding.(type) {
		case *types.GuardrailAutomatedReasoningFindingMemberValid:
			_ = v.Value // Value is types.GuardrailAutomatedReasoningFindingMemberValid
			fmt.Printf("VALID\n")
		case *types.GuardrailAutomatedReasoningFindingMemberInvalid:
			_ = v.Value // Value is types.GuardrailAutomatedReasoningInvalidFinding
			fmt.Printf("INVALID\n")
		case *types.GuardrailAutomatedReasoningFindingMemberImpossible:
			_ = v.Value // Value is types.GuardrailAutomatedReasoningImpossibleFinding
			fmt.Printf("IMPOSSIBLE\n")
		case *types.GuardrailAutomatedReasoningFindingMemberNoTranslations:
			_ = v.Value // Value is types.GuardrailAutomatedReasoningNoTranslationsFinding
			fmt.Printf("NO TRANSLATION\n")
		case *types.GuardrailAutomatedReasoningFindingMemberSatisfiable:
			_ = v.Value // Value is types.GuardrailAutomatedReasoningSatisfiableFinding
			fmt.Printf("SATISFIABLE\n")
		case *types.GuardrailAutomatedReasoningFindingMemberTooComplex:
			_ = v.Value // Value is types.GuardrailAutomatedReasoningTooComplexFinding
			fmt.Printf("TOO_COMPLEX\n")
		case *types.GuardrailAutomatedReasoningFindingMemberTranslationAmbiguous:
			_ = v.Value // Value is types.GuardrailAutomatedReasoningTranslationAmbiguousFinding
			fmt.Printf("TRANSLATION_AMBIGUOUS\n")
		case *types.UnknownUnionMember:
			fmt.Println("unknown tag:", v.Tag)

		default:
			fmt.Println("union is nil or unknown type")

		}
	}

	return string(resultJson)
}
