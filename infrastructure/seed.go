package infrastructure

import "github.com/nemdull/sf-meter-api/domain"

// SeedTechs は、最初から入れておく SF テクノロジーを返す。
// PostgreSQL 側の初期データ（db/init.sql）と同じ内容・同じ順番にしてある。
// 片方だけ直すとずれるので、seed_test.go が 2 つを突き合わせている。
func SeedTechs() []domain.Tech {
	return []domain.Tech{
		{Name: "ドミネーター", Description: "公安局特殊武器、サイコパス判定連動の支配射撃体系", Score: 35},
		{Name: "シビラシステム", Description: "脳スキャンで犯罪係数を算出する全国犯罪予防システム", Score: 91},
		{Name: "ニューラルリンク", Description: "脳とコンピュータを直接接続するBCIデバイス", Score: 82},
		{Name: "反重力エンジン", Description: "重力制御による無音・無排気の次世代推進技術", Score: 15},
		{Name: "ナノマシン", Description: "体内を自律修復する医療ナノロボット群", Score: 55},
		{Name: "フォトン魚雷", Description: "光子束を収束照射する艦載兵器", Score: 28},
		{Name: "ホログラム通信", Description: "立体映像によるリアルタイム遠距離コミュニケーション", Score: 72},
		{Name: "量子暗号通信", Description: "解読不可能な量子もつれを利用した秘匿通信", Score: 88},
		{Name: "コールドスリープ", Description: "長期宇宙航行向け代謝停止冬眠システム", Score: 44},
		{Name: "テレポーテーション", Description: "物体の空間転移を実現する瞬間移動装置", Score: 3},
		{Name: "タイムマシン", Description: "時空間を操作し過去・未来へ移動する装置", Score: 1},
		{Name: "バイオエンハンスメント", Description: "遺伝子編集による身体能力の恒久的強化", Score: 67},
		{Name: "エネルギーシールド", Description: "指向性エネルギー場による物理・熱防護バリア", Score: 41},
		{Name: "AI人格移植", Description: "人間の意識をデジタル基盤へ完全転写する技術", Score: 22},
		{Name: "マトリックス接続", Description: "仮想現実に意識を没入させるフルダイブインターフェース", Score: 59},
	}
}
