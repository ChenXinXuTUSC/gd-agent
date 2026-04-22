package main

import (
	"fmt"
	"gd-agent/pkg/llms"
	"gd-agent/pkg/ui"
	llm_types"gd-agent/pkg/llms/types"
)

func main() {
	state := llm_types.State{Stream: true}
	deepseek := llms.ProviderList["DeepSeek"]

	for {
		userInput := ""
		systemOutput := []rune{}
		fmt.Println(ui.UserLabel.Render("user") )
		fmt.Print("> ")
		fmt.Scanln(&userInput)
	
		state.Messages = append(state.Messages, llm_types.Message{
			Role: "user",
			Content: userInput,
		})
	
		runeCh, err := deepseek.GetResponse(&state)
		if err != nil {
			panic(err)
		}
	
		fmt.Println(ui.AssistantLabel.Render("assistant"))
		for r := range runeCh {
			systemOutput = append(systemOutput, r)
			fmt.Print(string(r))
		}
		fmt.Println()

		state.Messages = append(state.Messages, llm_types.Message{
			Role: "assistant",
			Content: string(systemOutput),
		})
	}


}
