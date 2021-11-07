package main

import (
	"bufio"
	"fmt"
	"math"
	"math/rand"
	"os"
	"sort"
	"strconv"
	"time"
)

const (
	DEBUG                    = true
	MIN_ESTIMATE_HISTORY_LEN = 10 //良さそうなのは30
	HC_LOOP_COUNT            = 50 //増やせばスコアは伸びるか？
)

var (
	day               int //現在日時
	N                 int
	M                 int
	K                 int
	R                 int
	u, v              int
	d                 [1000][20]int
	V                 [1000][]int   //依存関係を管理
	taskStatus        [1000]int     //タスクステータス管理, 0:not yet, 1:working, 2:done
	taskIsBookedBy    [1000]int     //タスクが誰に予約されているか
	memberStatus      [20]int       //メンバーステータス管理, 0:free, 1:working
	memberHistory     [20][]int     //タスク実行履歴
	memberEstimated   [20]int       //メンバーのスキル推定がされているかどうか
	memberBookingTask [20][]int     //メンバーが予約しているタスク一覧
	ps                [20][20]int   //メンバーのスキルの推定値
	psMin             [20][20]int   //メンバーのスキルの推定値の下限
	sTrue             [20][20]int   //メンバーのスキル(本物)
	tTrue             [1000][20]int //メンバーがタスクを処理するのにかかる時間(本物)
	taskStart         [1000]int     //タスクを開始した時刻
	taskEnd           [1000]int     //タスクを終了した時刻
	taskSize          [1000]int     //タスクの大きさ
	rank              [1000]int     //タスクの依存関係の深さ
	rank2             [1000]int     //タスクの依存関係の深さ2
	rank3             [1000]int     //タスクの依存関係の深さ3
	sMax              [20]int       //s_kの取りうる上限
	sortedTasks       []int         //rank順にソートされたタスク
	tmpScores         [1000]int     //一時計算用のテーブル

	allTimeEst time.Duration //推定にかかってる時間

)

func main() {
	fmt.Scanf("%d %d %d %d", &N, &M, &K, &R)
	for i := 0; i < N; i++ {
		for j := 0; j < K; j++ {
			fmt.Scanf("%d", &d[i][j])
			sMax[j] = max(sMax[j], d[i][j])
		}
		taskSize[i] = size(i)
		taskIsBookedBy[i] = -1
	}
	for i := 0; i < R; i++ {
		fmt.Scanf("%d %d", &u, &v)
		u -= 1 //indexを0に揃える
		v -= 1
		V[v] = append(V[v], u) //タスクvはタスクuに依存している
	}

	//デバッグ用, memberの真のスキルを読み込む
	if DEBUG {
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
	}

	//rank計算
	for t := 0; t < N; t++ { //初期化
		rank[t] = -1
	}
	for t := 0; t < N; t++ {
		calcRank(t, 0)
		calcRank3(t, taskSize[t])
		for _, u := range V[t] {
			rank2[u]++
		}
	}
	for t := 0; t < N; t++ { //rank表を表示
		fmt.Printf("# %d size = %d, rank = %d, rank2 = %d, rank3 = %d\n", t, taskSize[t], rank[t], rank2[t], rank3[t])
	}

	// rankが大きい順にtaskを並べておく(rankが大きい物はボトルネックになる)
	for t := 0; t < N; t++ {
		sortedTasks = append(sortedTasks, t)
	}
	sort.Slice(sortedTasks, func(i, j int) bool {
		a := sortedTasks[i]
		b := sortedTasks[j]
		return rank[a] > rank[b]
	})
	for _, t := range sortedTasks { //rank表を表示
		fmt.Printf("# %d rank = %d, rank2 = %d, size = %d\n", t, rank[t], rank2[t], taskSize[t])
	}

	var wtr = bufio.NewWriter(os.Stdout)
	var n int
	var f int
	for {
		fmt.Printf("#day %d\n", day)
		var nexta []int
		var nextb []int

		//処理したタスクが多い順にmemberをソートする, 優秀なメンバーはたくさんのタスクを処理しがち
		var sortedMembers []int
		for i := 0; i < M; i++ {
			sortedMembers = append(sortedMembers, i)
		}
		sort.Slice(sortedMembers, func(i, j int) bool {
			return len(memberHistory[sortedMembers[i]]) > len(memberHistory[sortedMembers[j]])
		})

		// 学習データが溜まったらパラメータを推定する
		// working中のメンバーであっても計算を行う, 何度も山登りすることで精度が上がる
		estimatedNum := 0
		for _, i := range sortedMembers {
			if len(memberHistory[i]) > MIN_ESTIMATE_HISTORY_LEN {
				//ここの数値は要調整, ある程度学習データがないと推定がかなり甘くなる
				estimate(i)
				memberEstimated[i] = 1
				estimatedNum++
			}
		}
		if estimatedNum == M {
			fmt.Printf("#all estimated\n")
		}

		//そもそもassign可能なタスクとassign可能なメンバーを洗い出す
		canAssignMemberNum := 0
		canAssignTaskNum := 0
		var canAssignTasks []int
		for _, i := range sortedMembers {
			if memberStatus[i] == 0 {
				canAssignMemberNum++
			}
		}
		for _, t := range sortedTasks {
			if canAssign(t) {
				canAssignTaskNum++
				canAssignTasks = append(canAssignTasks, t)
			}
		}
		fmt.Printf("#canAssign member=%d, task=%d\n", canAssignMemberNum, canAssignTaskNum)

		// ランク高いやつ上位7個から最適な割り当てを全探索する
		// if estimatedNum == M {
		if false {
			right := min(5, canAssignTaskNum)
			dfsTargetTasks = canAssignTasks[:right]
			dfsBestAssignMembersEndTime = 10000000
			for idx := range dfsTargetTasks {
				dfsAssignMembers[idx] = -1
				dfsBestAssignMembers[idx] = -1
			}
			for i := 0; i < M; i++ {
				if memberEstimated[i] == 0 {
					continue
				}
				for t := 0; t < N; t++ {
					if taskStatus[t] == 2 {
						continue
					}
					dfsTmpScores[i][t] = scoreTrue(ps[i], t)
				}
			}
			dfsFindBestAssignMembers(0)
			fmt.Printf("#bestAssign score=%d\n", dfsBestAssignMembersEndTime)
			for i := 0; i < len(dfsTargetTasks); i++ {
				fmt.Printf("#bestAssign task=%d, member=%d\n", dfsTargetTasks[i], dfsBestAssignMembers[i])
				m := dfsBestAssignMembers[i]
				t := dfsTargetTasks[i]
				if m == -1 {
					continue
				}
				memberBookingTask[m] = append(memberBookingTask[m], t)
				taskIsBookedBy[t] = m
			}
		}

		//通常の場合
		for _, i := range sortedMembers {
			if memberStatus[i] == 1 {
				continue
			}
			if len(memberBookingTask[i]) != 0 {
				//タスク予約中なので飛ばす
				continue
			}

			bestTask := findTask(i)

			if bestTask == -1 {
				continue
			}
			memberBookingTask[i] = append(memberBookingTask[i], bestTask)
			taskIsBookedBy[bestTask] = i
		}

		for m := 0; m < M; m++ {
			if memberStatus[m] == 1 {
				continue
			}
			if len(memberBookingTask[m]) == 0 {
				continue
			}
			t := memberBookingTask[m][0]
			memberBookingTask[m] = memberBookingTask[m][1:] //先頭をpop
			nexta = append(nexta, m)
			nextb = append(nextb, t)
			taskStatus[t] = 1
			memberStatus[m] = 1
			memberHistory[m] = append(memberHistory[m], t)
			taskStart[t] = day
		}

		fmt.Fprintf(wtr, "%d", len(nexta))
		for i := 0; i < len(nexta); i++ {
			fmt.Fprintf(wtr, " %d %d", nexta[i]+1, nextb[i]+1) //+1しておかないとインデックスがずれる
		}
		fmt.Fprintf(wtr, "\n")

		if DEBUG {
			error := 0
			n := 0
			for i := 0; i < M; i++ {
				fmt.Fprintf(wtr, "#s %d", i+1) //予測値を出力
				for k := 0; k < K; k++ {
					fmt.Fprintf(wtr, " %d", ps[i][k])
				}
				if memberEstimated[i] == 1 {
					n++
					for k := 0; k < K; k++ {
						error += (ps[i][k] - sTrue[i][k]) * (ps[i][k] - sTrue[i][k])
					}
				}
				fmt.Fprintf(wtr, "\n")
			}
			if n != 0 {
				error /= n
				fmt.Printf("#estimate error = %d\n", error)
			}
		}

		err := wtr.Flush() //flushしないとだめ
		if err != nil {
			fmt.Printf("error %s\n", err.Error())
			os.Exit(1)
		}

		day++

		if DEBUG {
			fmt.Printf("#allTimeEst = %fs\n", allTimeEst.Seconds())
		}

		fmt.Scanf("%d", &n)
		if n == -1 {
			if DEBUG {
				err := writeEstError()
				if err != nil {
					fmt.Printf("error %s\n", err.Error())
					os.Exit(1)
				}
			}
			break
		}
		for i := 0; i < n; i++ {
			fmt.Scanf("%d", &f)
			f -= 1 //indexを0に揃える
			memberStatus[f] = 0
			t := memberHistory[f][len(memberHistory[f])-1]
			taskStatus[t] = 2 //taskをdoneに
			taskEnd[t] = day

			//パラメータの下限が確定(下振れを考慮)
			actDay := taskEnd[t] - taskStart[t]
			for k := 0; k < K; k++ {
				psMin[f][k] = max(psMin[f][k], d[t][k]-actDay-3)
				ps[f][k] = max(psMin[f][k], ps[f][k])
			}
		}
	}
}

var dfsTmpScores [20][1000]int
var dfsTargetTasks []int
var dfsAssignMembers [10]int
var dfsBestAssignMembers [10]int
var dfsBestAssignMembersEndTime int //時間だけだと時間内のタスクのアサインが適当になるかも
func dfsFindBestAssignMembers(depth int) {
	if depth == len(dfsTargetTasks) {
		//計算処理
		maxScore := day
		for m := 0; m < M; m++ {
			if memberEstimated[m] == 0 { //推定されていない場合スキップ
				continue
			}
			score := day
			if memberStatus[m] == 1 {
				working := memberHistory[m][len(memberHistory[m])-1]
				endTime := taskStart[working] + dfsTmpScores[m][working] + 3 //上振れも考慮する?
				score = endTime
			}
			mUse := false
			for i := 0; i < len(dfsTargetTasks); i++ {
				m2 := dfsAssignMembers[i]
				if m != m2 {
					continue
				}
				mUse = true
				score += dfsTmpScores[m][dfsTargetTasks[i]] + 3 //上振れも考慮する?
			}
			if mUse {
				maxScore = max(maxScore, score)
			}
		}
		if maxScore < dfsBestAssignMembersEndTime {
			dfsBestAssignMembersEndTime = maxScore
			for i := 0; i < len(dfsTargetTasks); i++ {
				dfsBestAssignMembers[i] = dfsAssignMembers[i]
			}
		}
		return
	}
	for i := 0; i < M; i++ {
		if memberEstimated[i] == 0 { //推定されていない場合スキップ
			continue
		}
		if len(memberBookingTask[i]) != 0 {
			//タスクを予約しているので探索に含めない
			continue
		}
		if dfsBestAssignMembersEndTime < day+dfsTmpScores[i][dfsTargetTasks[depth]]+3 { //上振れも考慮する?
			continue
		}
		dfsAssignMembers[depth] = i
		dfsFindBestAssignMembers(depth + 1)
		dfsAssignMembers[depth] = -1
	}
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
			if bestRank == -1 {
				bestRank = rank[t]
			}
			//スキルが推定されている場合はrankが同じやつリストを一旦作る
			if bestRank <= rank[t] {
				targets = append(targets, t)
				tmpScores[t] = scoreTrue(ps[member], t)
			}
		} else {
			//スキルが推定されていない場合はrankが高い順に処理
			bestTask = t
			return bestTask
		}
	}

	if memberEstimated[member] == 1 {
		fmt.Printf("#len(targets) = %d\n", len(targets))
	}

	for _, t := range targets {
		if bestTask == -1 {
			bestTask = t
			continue
		}
		if rank3[bestTask] == rank3[t] {
			if tmpScores[t] < tmpScores[bestTask] {
				bestTask = t
				continue
			}
		} else if rank3[bestTask] < rank3[t] {
			bestTask = t
		}
		// if tmpScores[t] == tmpScores[bestTask] {
		// 	// if taskSize[bestTask] < taskSize[t] { //スコアが同じ場合はより重たいもの
		// 	// 	bestTask = t
		// 	// 	continue
		// 	// }
		// 	if rank3[bestTask] < rank3[t] { //スコアが同じ場合はrank3が大きいもの
		// 		bestTask = t
		// 		continue
		// 	}
		// } else if tmpScores[t] < tmpScores[bestTask] { //スコアが低い方優先
		// 	bestTask = t
		// }
	}

	if bestTask != -1 {
		for i := 0; i < M; i++ {
			if memberStatus[i] == 1 || i == member {
				continue
			}
			if memberEstimated[i] == 0 {
				continue
			}
			if len(memberBookingTask[i]) != 0 {
				continue
			}
			//自分以外で最適な人がいるか確認
			score := scoreTrue(ps[i], bestTask)
			if score < tmpScores[bestTask] {
				fmt.Printf("#better %d %d\n", tmpScores[bestTask], score)
				//いるのでやらない
				return -1
			}
		}
		bestWaitMember := -1
		bestWaitTimeDiff := 0
		for i := 0; i < M; i++ { //workingメンバー用の処理
			if memberStatus[i] != 1 || i == member {
				continue
			}
			if memberEstimated[i] == 0 {
				continue
			}
			//自分以外で最適な人がいるか確認
			score := day + calcWaitTime(i) + scoreTrue(ps[i], bestTask)
			if score < day+tmpScores[bestTask] {
				diff := day + tmpScores[bestTask] - score
				if bestWaitTimeDiff < diff {
					bestWaitMember = i
					bestWaitTimeDiff = diff
				}
				fmt.Printf("#god task = %d, beforeMember = %d, godMember = %d, endDay = %d, before = %d, diff = %d\n", bestTask, member, i, score, day+tmpScores[bestTask], day+tmpScores[bestTask]-score)
				//いるらしい
			}
		}
		if bestWaitMember != -1 {
			fmt.Printf("#final god task = %d, beforeMember = %d, godMember = %d, bestDiff = %d\n", bestTask, member, bestWaitMember, bestWaitTimeDiff)
			if bestWaitTimeDiff > 10 {
				// かなり良さそうなので予約する
				memberBookingTask[bestWaitMember] = append(memberBookingTask[bestWaitMember], bestTask)
				taskIsBookedBy[bestTask] = bestWaitMember
				return -1
			}
		}
	}

	return bestTask
}

func calcWaitTime(member int) int { //そのメンバーの再アサインが可能になるまで最短でどれぐらいの時間がかかるか
	ret := day //いつ終わるか
	if memberStatus[member] == 1 {
		working := memberHistory[member][len(memberHistory[member])-1]
		tmp := scoreTrue(ps[member], working)
		var endTime int
		if tmp == 1 {
			endTime = taskStart[working] + tmp
		} else {
			endTime = taskStart[working] + tmp + 3 //上振れも考慮する?
		}

		ret = endTime
	}
	for _, t := range memberBookingTask[member] {
		tmp := scoreTrue(ps[member], t)
		var endTime int
		if tmp == 1 {
			endTime = taskStart[t] + tmp
		} else {
			endTime = taskStart[t] + tmp + 3 //上振れも考慮する?
		}
		ret += endTime
	}
	return ret - day
}

func canAssign(task int) bool {
	if taskStatus[task] != 0 {
		//タスクが終わったか誰かやってる
		return false
	}
	if taskIsBookedBy[task] != -1 {
		//タスクが予約済み
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
// 山登り法により推定する
func estimate(member int) {
	startTime := time.Now()
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
	var targetK2 int
	var add bool
	var error int
	var success bool
	l := 0
	for {
		targetK = rand.Intn(K)
		targetK2 = targetK
		if rand.Intn(3) == 0 {
			targetK2 = rand.Intn(K)
		}
		add = rand.Intn(2) == 0
		if add {
			now[targetK] = min(sMax[targetK], now[targetK]+1)
			if targetK2 != targetK {
				now[targetK2] = max(psMin[member][targetK2], now[targetK2]-1)
			}
		} else {
			now[targetK] = max(psMin[member][targetK], now[targetK]-1)
			if targetK2 != targetK {
				now[targetK2] = min(sMax[targetK2], now[targetK2]+1)
			}
		}

		success = false
		error = calcError(now, member)
		if bestError == error {
			if skillSize(now) < skillSize(bestSkill) { //エラーが同じ場合はskillがより小規模なもの
				success = true
			}
			// success = true
		} else if error < bestError {
			success = true
		}
		if success {
			bestError = error
			for k := 0; k < K; k++ {
				bestSkill[k] = now[k]
			}
		} else { //巻き戻す
			if add {
				now[targetK] = max(psMin[member][targetK], now[targetK]-1)
				if targetK2 != targetK {
					now[targetK2] = min(sMax[targetK2], now[targetK2]+1)
				}
			} else {
				now[targetK] = min(sMax[targetK], now[targetK]+1)
				if targetK2 != targetK {
					now[targetK2] = max(psMin[member][targetK2], now[targetK2]-1)
				}
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
	allTimeEst += time.Now().Sub(startTime)
}

func calcError(skill [20]int, member int) int {
	error := 0
	for _, t := range memberHistory[member] {
		if taskStatus[t] != 2 {
			continue
		}
		//今までに実行した全てのタスクから二乗誤差を算出
		si := scoreTrue(skill, t)
		ti := taskEnd[t] - taskStart[t]
		error += (si - ti) * (si - ti)
	}
	return error
}

func calcError2(skill [20]int, member int) int {
	error := 0
	for _, t := range memberHistory[member] {
		if taskStatus[t] != 2 {
			continue
		}
		//今までに実行した全てのタスクから絶対値誤差を算出
		si := scoreTrue(skill, t)
		ti := taskEnd[t] - taskStart[t]
		// error += (si - ti) * (si - ti) / 1000
		error += int(math.Abs(float64(si - ti)))
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

func calcRank3(task int, cost int) {
	if cost < rank3[task] {
		//計算済みのrankの方が上の場合無駄なので省略
		return
	}
	rank3[task] = cost

	next := V[task]
	for _, nextT := range next {
		calcRank3(nextT, cost+taskSize[nextT])
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

func writeEstError() error {
	error := 0
	n := 0
	for i := 0; i < M; i++ {
		if memberEstimated[i] == 1 {
			n++
			for k := 0; k < K; k++ {
				error += (ps[i][k] - sTrue[i][k]) * (ps[i][k] - sTrue[i][k])
			}
		}
	}
	if n == 0 {
		return nil
	}
	error /= n

	file, err := os.Create("./estscore.txt")
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = file.WriteString(strconv.Itoa(error) + "\n")
	if err != nil {
		return err
	}

	return nil
}
