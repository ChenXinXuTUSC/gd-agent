package main

import (
	"fmt"
	"gd-agent/pkg/llms"
	llm_types"gd-agent/pkg/llms/types"
)

func main() {
	state := llm_types.State{Stream: true}
	deepseek := llms.ProviderList["DeepSeek"]

	for {
		userInput := ""
		systemOutput := []rune{}
		fmt.Print("user: ")
		fmt.Scanln(&userInput)
	
		state.Messages = append(state.Messages, llm_types.Message{
			Role: "user",
			Content: userInput,
		})
	
		runeCh, err := deepseek.GetResponse(&state)
		if err != nil {
			panic(err)
		}
	
		fmt.Print("system: ")
		for r := range runeCh {
			systemOutput = append(systemOutput, r)
			fmt.Print(string(r))
		}

		state.Messages = append(state.Messages, llm_types.Message{
			Role: "system",
			Content: string(systemOutput),
		})
	}


}
