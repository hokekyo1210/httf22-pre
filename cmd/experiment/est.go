package main

import (
	"fmt"
	"math/rand"
	"time"
)

const (
	MIN_ESTIMATE_HISTORY_LEN = 25  //良さそうなのは30
	HC_LOOP_COUNT            = 100 //増やせばスコアは伸びるか？
)

var (
	N               int
	M               int
	K               int
	R               int
	u, v            int
	d               [1000][20]int
	V               [1000][]int   //依存関係を管理
	taskStatus      [1000]int     //タスクステータス管理, 0:not yet, 1:working, 2:done
	memberStatus    [20]int       //メンバーステータス管理, 0:free, 1:working
	memberHistory   [20][]int     //タスク実行履歴
	memberEstimated [20]int       //メンバーのスキル推定がされているかどうか
	ps              [20][20]int   //メンバーのスキルの推定値
	psMin           [20][20]int   //メンバーのスキルの推定値の下限
	sTrue           [20][20]int   //メンバーのスキル(本物)
	tTrue           [1000][20]int //メンバーがタスクを処理するのにかかる時間(本物)
	taskStart       [1000]int     //タスクを開始した時刻
	taskEnd         [1000]int     //タスクを終了した時刻
	taskSize        [1000]int     //タスクの大きさ
	rank            [1000]int     //タスクの依存関係の深さ
	sMax            [20]int       //s_kの取りうる上限
	sortedTasks     []int         //rank順にソートされたタスク
	tmpScores       [1000]int     //一時計算用のテーブル
)

func main() {
	rand.Seed(time.Now().UnixNano())
	fmt.Scanf("%d %d %d %d", &N, &M, &K, &R)
	for i := 0; i < N; i++ {
		for j := 0; j < K; j++ {
			fmt.Scanf("%d", &d[i][j])
			sMax[j] = max(sMax[j], d[i][j])
		}
		taskSize[i] = size(i)
	}
	for i := 0; i < R; i++ {
		fmt.Scanf("%d %d", &u, &v)
		u -= 1 //indexを0に揃える
		v -= 1
		V[v] = append(V[v], u) //タスクvはタスクuに依存している
	}

	fmt.Printf("#smax = %v\n", sMax)
	for i := 0; i < M; i++ { //初期解を0でなくsmaxからスタート
		for k := 0; k < K; k++ {
			ps[i][k] = sMax[k]
		}
	}

	//デバッグ用, memberの真のスキルとメンバーがタスクを実行するのにかかる時間を読み込む
	for i := 0; i < M; i++ {
		for k := 0; k < K; k++ {
			fmt.Scanf("%d", &sTrue[i][k])
			if sTrue[i][k] > sMax[k] {
				sTrue[i][k] = sMax[k]
			}
		}
	}

	for t := 0; t < N; t++ {
		for i := 0; i < M; i++ {
			fmt.Scanf("%d", &tTrue[t][i])
		}
	}

	m := 2
	limit := 100
	l := 0
	for {
		nt := rand.Intn(N)
		if !canAssign(nt) {
			continue
		}
		taskStatus[nt] = 2 //完了扱い

		memberHistory[m] = append(memberHistory[m], nt)
		taskStart[nt] = 0
		taskEnd[nt] = tTrue[nt][m]
		// if taskEnd[nt]-taskStart[nt] == 1 {
		// 	fmt.Printf("#1\n")
		// 	for k := 0; k < K; k++ {
		// 		psMin[m][k] = max(psMin[m][k], d[nt][k]-3) //下限が確定
		// 		ps[m][k] = max(psMin[m][k], d[nt][k])
		// 	}
		// }
		actDay := taskEnd[nt] - taskStart[nt]
		for k := 0; k < K; k++ {
			psMin[m][k] = max(psMin[m][k], d[nt][k]-actDay-3) //下限が確定(下振れを考慮)
			ps[m][k] = max(psMin[m][k], ps[m][k])
		}

		if l >= MIN_ESTIMATE_HISTORY_LEN {
			estimate(m)
		}

		//真のスキルとのdiffを確認
		diff := 0
		for k := 0; k < K; k++ {
			diff += (ps[m][k] - sTrue[m][k]) * (ps[m][k] - sTrue[m][k])
		}
		if l%10 == 0 {
			// trueError := calcError(sTrue[m], m)
			// actError := calcError(ps[m], m)
			fmt.Printf("#n = %d diff = %d\n", l, diff)
			// for k := 0; k < K; k++ {
			// 	fmt.Printf("#psMin[m][%d] = %d, sTrue[m][%d] = %d, ps[m][%d] = %d\n", k, psMin[m][k], k, sTrue[m][k], k, ps[m][k])
			// }
			// fmt.Printf("#trueError = %d, actError = %d\n", trueError, actError)
		}
		if l == limit {
			break
		}
		l++
	}

	// trueError := calcError(sTrue[m], m)
	// bef := calcError(ps[m], m)
	// // for v := 0; v <= 23; v++ {
	// // 	ps[m][11] = v
	// // 	af := calcError(ps[m], m)
	// // 	fmt.Printf("#true = %d, bef = %d, af(%d) = %d\n", trueError, bef, v, af)
	// // }
	// // ps[m][0] = 4
	// // ps[m][2] = 7
	// ps[m][3] = 4
	// ps[m][7] = 2
	// // ps[m][12] = 3

	// ps[m][1] = 13
	// ps[m][5] = 18
	// ps[m][8] = 20
	// ps[m][9] = 23
	// ps[m][10] = 11
	// ps[m][11] = 18
	// ps[m][13] = 15
	// ps[m][14] = 11
	// af := calcError(ps[m], m)
	// fmt.Printf("#true = %d, bef = %d, af = %d\n", trueError, bef, af)

}

func findTask(member int) int { //最適なタスクを選定する
	bestTask := -1
	bestRank := -1

	var targets []int

	//終了していないタスクの中から最適なタスクにアサインする
	//rankが高い順に処理されることに注意(sortedTasksが既にrank順でソート済み)
	for _, t := range sortedTasks {
		if !canAssign(t) {
			continue
		}
		if memberEstimated[member] == 1 {
			//スキルが推定されている場合はrankが同じやつリストを一旦作る
			if bestRank <= rank[t] {
				targets = append(targets, t)
				tmpScores[t] = scoreTrue(ps[member], t)
				bestRank = rank[t]
			}
		} else {
			//スキルが推定されていない場合はrankが高い順に処理
			bestTask = t
			break
		}
	}

	for _, t := range targets {
		if bestTask == -1 {
			bestTask = t
			continue
		}
		if tmpScores[t] == tmpScores[bestTask] {
			if taskSize[bestTask] < taskSize[t] { //スコアが同じ場合はより重たいもの
				bestTask = t
				continue
			}
		} else if tmpScores[t] < tmpScores[bestTask] { //スコアが低い方優先
			bestTask = t
		}
	}

	return bestTask
}

func canAssign(task int) bool {
	if taskStatus[task] != 0 {
		//タスクが終わったか誰かやってる
		return false
	}

	//依存するタスクがあって、そちらが終わってない場合は実行不可能
	canAssign := true
	for k := 0; k < len(V[task]); k++ {
		if taskStatus[V[task][k]] != 2 {
			canAssign = false
		}
	}
	return canAssign
}

// 山登り法により推定する
func estimate(member int) {
	bestError := 10000000000
	var bestSkill [20]int
	var now [20]int
	for k := 0; k < K; k++ {
		//最初は前回の推定結果を使う
		now[k] = ps[member][k]
		bestSkill[k] = ps[member][k]
	}
	bestError = calcError(bestSkill, member)

	var targetK int
	var add bool
	var error int
	var success bool
	l := 0
	for {
		targetK = rand.Intn(K)
		add = rand.Intn(2) == 0
		if add {
			now[targetK] = min(sMax[targetK], now[targetK]+1)
		} else {
			now[targetK] = max(psMin[member][targetK], now[targetK]-1)
		}

		success = false
		error = calcError(now, member)
		if bestError == error {
			if skillSize(now) < skillSize(bestSkill) { //エラーが同じ場合はskillがより小規模なもの
				success = true
			}
		} else if error < bestError {
			success = true
		} else if bestError < error { //悪くなる場合
			//学習データが少ないうちは悪い方にも確率で行ってみる
			p := rand.Float64()
			prob := prob(bestError, error, len(memberHistory[member]))
			// fmt.Printf("#%d %d %f\n", bestError, error, prob)
			if p < prob {
				success = true
			}
		}
		if success {
			bestError = error
			for k := 0; k < K; k++ {
				bestSkill[k] = now[k]
			}
		} else { //巻き戻す
			if add {
				now[targetK] = max(psMin[member][targetK], now[targetK]-1)
			} else {
				now[targetK] = min(sMax[targetK], now[targetK]+1)
			}
		}

		l++
		if l == HC_LOOP_COUNT { //最終的にはタイマーにしたい
			break
		}
	}

	for k := 0; k < K; k++ {
		ps[member][k] = bestSkill[k]
	}
}

func prob(before int, after int, n int) float64 {
	return 0.0
	// return float64(100-n) / 100.0
	// return math.Exp(float64((before - after) / (101 - n)))
}

func calcError(skill [20]int, member int) int {
	error := 0
	for _, t := range memberHistory[member] {
		//今までに実行した全てのタスクから二乗誤差を算出
		si := scoreTrue(skill, t)
		ti := taskEnd[t] - taskStart[t]
		// if ti != 1 && si > 5 {
		// 	continue
		// }
		error += (si - ti) * (si - ti)
		// error += int(math.Abs(float64(si - ti)))
	}
	return error
}

func calcRank(task int, depth int) {
	if depth < rank[task] {
		//計算済みのrankの方が上の場合無駄なので省略
		return
	}
	rank[task] = depth

	next := V[task]
	for _, nextT := range next {
		calcRank(nextT, depth+1)
	}
}

func scoreTrue(skill [20]int, task int) int {
	score := 0
	for k := 0; k < K; k++ {
		score += max(0, d[task][k]-skill[k])
	}
	return max(1, score)
}

func max(a int, b int) int {
	if a < b {
		return b
	}
	return a
}

func min(a int, b int) int {
	if a < b {
		return a
	}
	return b
}

func size(task int) int {
	sum := 0
	for k := 0; k < K; k++ {
		sum += d[task][k]
	}
	return sum
}

func skillSize(skill [20]int) int {
	sum := 0
	for k := 0; k < K; k++ {
		sum += skill[k]
	}
	return sum
}
