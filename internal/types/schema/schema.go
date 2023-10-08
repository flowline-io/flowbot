package schema

import (
	"github.com/flowline-io/flowbot/internal/types"
)

func Stage(stages ...types.Stage) []types.Stage {
	return stages
}

func Action(id string) types.Stage {
	return types.Stage{
		Type: types.ActionStage,
		Flag: id,
	}
}

func Bot(name string) types.Bot {
	return types.Bot(name)
}

func Command(bot types.Bot, token ...string) types.Stage {
	return types.Stage{
		Type: types.CommandStage,
		Bot:  bot,
		Args: token,
	}
}

func Form(id string) types.Stage {
	return types.Stage{
		Type: types.FormStage,
		Flag: id,
	}
}

func Instruct(id string, args ...string) types.Stage {
	return types.Stage{
		Type: types.InstructStage,
		Flag: id,
		Args: args,
	}
}

func Session(id string, args ...string) types.Stage {
	return types.Stage{
		Type: types.SessionStage,
		Flag: id,
		Args: args,
	}
}

func CommandTrigger(define string) types.Trigger {
	return types.Trigger{
		Type:   types.TriggerCommandType,
		Define: define,
	}
}
