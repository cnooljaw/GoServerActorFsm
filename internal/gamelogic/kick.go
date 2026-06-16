package gamelogic

const baseShrewReward = 10

type KickInput struct {
	HammerType int
	ComboID    int
	Shrews     []KickShrew
}

type KickShrew struct {
	Index       int
	ProtectType int
}

type KickResult struct {
	Money        int
	Angry        int
	Power        int
	LevelScore   int
	HammerID     int
	NumOfShrew   int
	ShrewRewards []ShrewReward
	Combo        int
	ComboID      int
}

type ShrewReward struct {
	ShrewIndex int
	Reward     int
}

func CalculateKickResult(input KickInput) KickResult {
	rewards := make([]ShrewReward, 0, len(input.Shrews))
	totalReward := 0

	for _, shrew := range input.Shrews {
		reward := baseShrewReward
		totalReward += reward
		rewards = append(rewards, ShrewReward{
			ShrewIndex: shrew.Index,
			Reward:     reward,
		})
	}

	return KickResult{
		Money:        totalReward,
		LevelScore:   totalReward,
		HammerID:     input.HammerType,
		NumOfShrew:   len(input.Shrews),
		ShrewRewards: rewards,
		Combo:        len(input.Shrews),
		ComboID:      input.ComboID,
	}
}
