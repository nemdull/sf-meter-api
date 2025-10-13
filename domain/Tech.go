package domain

type Tech struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Score       int    `json:"score"`
}

// とりあえずモックデータ
var techs = []Tech{
	{ID: 1, Name: "ドミネーター", Description: "犯罪係数で裁く未来銃", Score: 35},
	{ID: 2, Name: "タイムマシン", Description: "時間を超える装置", Score: 5},
	{ID: 3, Name: "量子通信", Description: "瞬時に情報伝送（量子もつれ）", Score: 60},
}
