// est_greedy_speed2.goをベースにしている.
// メンバーごとの推定精度の上がり型を確認する。(LEN=6で精度が良い理由も確認する)

package main

import (
	"bufio"
	"fmt"
	"math/rand"
	"os"
	"sort"
	"strconv"
	"time"
)

const (
	DEBUG                    = true
	MIN_ESTIMATE_HISTORY_LEN = 0  //良さそうなのは30
	HC_LOOP_COUNT            = 50 //増やせばスコアは伸びるか？ あまり変える余地ないかも
	FREE_MARGIN              = 6  //a
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
	sMax              [20]int       //s_kの取りうる値の上限
	sortedTasks       []int         //rank順にソートされたタスク
	sortedTasks2      []int         //tasksize順にソートされたタスク

	allTimeEst         time.Duration //推定にかかってる時間
	allTimeSearch      time.Duration //探索にかかってる時間
	allTimeSearchCalc1 time.Duration
	allTimeSearchCalc2 time.Duration
	allTime            time.Duration //全体にかかっている時間

	estimateHistory [20][]EstimateHistory //推定の結果履歴
)

type EstimateHistory struct {
	bestError        int
	bestError2       int
	bestTrueError    int
	trueError        int
	memberHistoryNum int
	transitionNum    int
}

func main() {
	startAllTime := time.Now()
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
		calcRank2(t, taskSize[t])
	}
	if DEBUG {
		for t := 0; t < N; t++ { //rank表を表示
			fmt.Printf("# %d size = %d, rank = %d, rank2 = %d\n", t, taskSize[t], rank[t], rank2[t])
		}
	}

	// rankが大きい順にtaskを並べておく(rankが大きい物はボトルネックになる)
	for t := 0; t < N; t++ {
		sortedTasks = append(sortedTasks, t)
		sortedTasks2 = append(sortedTasks2, t)
	}
	sort.Slice(sortedTasks, func(i, j int) bool {
		a := sortedTasks[i]
		b := sortedTasks[j]
		return rank2[a] > rank2[b]
		// if rank[a] == rank[b] {
		// 	return rank2[a] > rank2[b] //rankが同じ場合はrank2優先
		// return taskSize[a] < taskSize[b]
		// }
		// return rank[a] > rank[b]
	})
	sort.Slice(sortedTasks2, func(i, j int) bool {
		a := sortedTasks2[i]
		b := sortedTasks2[j]
		// return rank2[a] > rank2[b]
		if rank[a] == rank[b] {
			// 	return rank2[a] > rank2[b] //rankが同じ場合はrank2優先
			return taskSize[a] < taskSize[b]
		}
		return rank[a] > rank[b]
	})
	if DEBUG {
		for _, t := range sortedTasks { //rank表を表示
			fmt.Printf("# %d rank = %d, size = %d\n", t, rank[t], taskSize[t])
		}
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
			working := -1
			if memberStatus[i] == 1 {
				working = memberHistory[i][len(memberHistory[i])-1]
			}
			fmt.Printf("#member = %d, memberStatus = %d, taskLen = %d, working = %d\n", i, memberStatus[i], len(memberHistory[i]), working)
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
				for k := 0; k < K; k++ {
					ps[i][k] = sTrue[i][k]
				}
			}
		}
		if estimatedNum == M {
			fmt.Printf("#all estimated\n")
		}

		//そもそもassign可能なタスクとassign可能なメンバーを洗い出す
		canAssignMemberNum := 0
		canAssignTaskNum := 0
		for _, i := range sortedMembers {
			if memberStatus[i] == 0 {
				canAssignMemberNum++
			}
		}
		for _, t := range sortedTasks {
			if canAssign(t, false) {
				canAssignTaskNum++
			}
		}
		fmt.Printf("#canAssign member=%d, task=%d\n", canAssignMemberNum, canAssignTaskNum)

		//実験中
		if estimatedNum == M && canAssignMemberNum != 0 {
			searchStart := time.Now()
			experiment()
			allTimeSearch += time.Now().Sub(searchStart)
		} else {

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
		}

		for m := 0; m < M; m++ {
			if memberStatus[m] == 1 {
				continue
			}
			if len(memberBookingTask[m]) == 0 {
				continue
			}
			t := memberBookingTask[m][0]
			if !canAssign(t, false) {
				continue
			}
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
			allTime = time.Now().Sub(startAllTime)
			fmt.Printf("#allTime = %fs\n", allTime.Seconds())
			fmt.Printf("#allTimeEst = %fs\n", allTimeEst.Seconds())
			fmt.Printf("#allTimeSearch = %fs\n", allTimeSearch.Seconds())
			fmt.Printf("#allTimeSearchCalc1 = %fs\n", allTimeSearchCalc1.Seconds())
			fmt.Printf("#allTimeSearchCalc2 = %fs\n", allTimeSearchCalc2.Seconds())
		}

		fmt.Scanf("%d", &n)
		if n == -1 {
			if DEBUG {
				err := writeEstError()
				if err != nil {
					fmt.Printf("error %s\n", err.Error())
					os.Exit(1)
				}

				err = writeEstHistory()
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

			trueDay := taskEnd[t] - taskStart[t]
			estimateDay := scoreTrue(ps[f], t)

			if 100 < day && abs(trueDay-estimateDay) > 10 {
				fmt.Printf("# check member = %d, task = %d, trueDay = %d, estimateDay = %d\n", f, t, trueDay, estimateDay)
				for k := 0; k < K; k++ {
					ps[f][k] = 0
				}
				for l := 0; l < 10; l++ {
					estimate(f)
				}
			}

			//パラメータの下限が確定(下振れを考慮)
			actDay := taskEnd[t] - taskStart[t]
			for k := 0; k < K; k++ {
				psMin[f][k] = max(psMin[f][k], d[t][k]-actDay-3)
				ps[f][k] = max(psMin[f][k], ps[f][k])
			}
		}
	}
}

func experiment() {
	searchCalc1Start := time.Now()
	//全タスクに対するscoreの総量を全員分計算してみる
	// skill := sTrue
	skill := ps

	var taskScoreMin [1000]int
	var taskScoreMinMember [1000]int
	var scoreAll [20]int
	var tmpScoreAll [20][1000]int

	for t := 0; t < N; t++ {
		if taskStatus[t] != 0 { //未実行タスクのみを対象
			continue
		}
		taskScoreMinMember[t] = -1
		taskScoreMin[t] = 100000000
		rank3[t] = -1
	}
	var membersRanking []int
	for m := 0; m < M; m++ {
		membersRanking = append(membersRanking, m)
		for t := 0; t < N; t++ {
			if taskStatus[t] != 0 { //未実行タスクのみを対象
				continue
			}
			tmpScoreAll[m][t] = scoreTrue(skill[m], t)
			taskScoreMin[t] = min(taskScoreMin[t], tmpScoreAll[m][t])
			scoreAll[m] += tmpScoreAll[m][t]
		}
		// fmt.Printf("#member = %d, scoreAll = %d\n", m, scoreAll[m])
	}

	// memberをscoreALL小さい順に並べる
	sort.Slice(membersRanking, func(i, j int) bool {
		return scoreAll[membersRanking[i]] < scoreAll[membersRanking[j]]
	})

	// fmt.Printf("#ranking = %v\n", membersRanking)

	for i := len(membersRanking) - 1; i != -1; i-- {
		m := membersRanking[i]
		for t := 0; t < N; t++ {
			if taskStatus[t] != 0 { //未実行タスクのみを対象
				continue
			}
			score := tmpScoreAll[m][t]
			if score == taskScoreMin[t] && taskScoreMinMember[t] == -1 {
				taskScoreMinMember[t] = m
			}
		}
	}

	// rank3を計算
	for i := len(sortedTasks) - 1; i != 0; i-- {
		t := sortedTasks[i]
		if taskStatus[t] != 0 {
			continue
		}
		m := taskScoreMinMember[t]
		score := max(1, tmpScoreAll[m][t]-3)
		calcRank3(skill, taskScoreMinMember, t, score)
	}

	// sort.Slice(sortedTasks, func(i, j int) bool {
	// 	a := sortedTasks[i]
	// 	b := sortedTasks[j]
	// 	if rank3[a] == rank3[b] {
	// 		return rank[a] > rank[b]
	// 	}
	// 	return rank3[a] > rank3[b]
	// })

	for _, t := range sortedTasks { //rank表を表示
		if taskStatus[t] != 0 {
			continue
		}
		fmt.Printf("# %d rank = %d, rank2 = %d, rank3 = %d, size = %d, bestScore = %d, canAssign = %v\n", t, rank[t], rank2[t], rank3[t], taskSize[t], tmpScoreAll[taskScoreMinMember[t]][t], canAssign(t, false))
	}

	// 一番得意なメンバーをassign
	for m := 0; m < M; m++ {
		memberBookingTask[m] = make([]int, 0)
	}
	for _, t := range sortedTasks {
		if taskStatus[t] != 0 {
			continue
		}
		// fmt.Printf("#task = %d, Max = %d, Avg = %d, Min = %d, who = %d, rank = %d\n", t, taskScoreMax[t], taskScoreAvg[t], taskScoreMin[t], taskScoreMinMember[t], rank[t])
		m := taskScoreMinMember[t]
		if m == -1 {
			fmt.Printf("# its a bug!\n")
			continue
		}
		memberBookingTask[m] = append(memberBookingTask[m], t)
		taskIsBookedBy[t] = m
	}

	for m := 0; m < M; m++ {
		fmt.Printf("#member = %d, memberBookingTask = %v\n", m, memberBookingTask[m])
	}

	// 次のタスクまでの間が十分長い人の中から、最も早くタスクを終えられる人を探してassign
	var remainMember []int
	for m := 0; m < M; m++ {
		if memberStatus[m] == 1 {
			continue
		}
		if len(memberBookingTask[m]) != 0 {
			nextT := memberBookingTask[m][0]
			if canAssign(nextT, false) { //今すぐにassign出来るタスクを抱えているのでこのメンバーは除外
				// fmt.Printf("#blocked\n")
				continue
			}
		}
		remainMember = append(remainMember, m)
	}
	for t := 0; t < N; t++ {
		memo[t] = -1 //メモを初期化
	}

	allTimeSearchCalc1 += time.Now().Sub(searchCalc1Start)
	searchCalc2Start := time.Now()

	for _, t := range sortedTasks {
		if len(remainMember) == 0 { //全員assignされたら一応終わる
			break
		}
		if !canAssign(t, false) { //今すぐにassign出来るタスクのみを対象
			continue
		}
		memberIsBooking := taskIsBookedBy[t] //タスクを予約している人
		if memberStatus[memberIsBooking] == 0 && memberBookingTask[memberIsBooking][0] == t {
			// 今日中に実行予定のメンバーがいるのでスキップ
			continue
		}

		trueEndTime := day + calcWaitTime(memberIsBooking) //本来このタスクが終わる時間
		for _, bookedT := range memberBookingTask[memberIsBooking] {
			trueEndTime += tmpScoreAll[memberIsBooking][bookedT]
			if bookedT == t {
				break
			}
		}
		// fmt.Printf("#task = %d, memberIsBooking = %d, trueEndTime = %d, rank = %d, trueEndTime = %d\n", t, memberIsBooking, trueEndTime, rank[t], trueEndTime)

		bestEndTime := 10000000000
		bestMember := -1
		bestDeadline := 10000000000
		for m := 0; m < M; m++ {
			if memberIsBooking == m {
				continue
			}
			if len(memberBookingTask[m]) != 0 {
				nextT := memberBookingTask[m][0]
				if canAssign(nextT, false) { //今すぐにassign出来るタスクを抱えているのでこのメンバーは除外
					// fmt.Printf("#blocked\n")
					continue
				}
			}
			freeTime := 1000000000
			if len(memberBookingTask[m]) != 0 {
				nextTask := memberBookingTask[m][0] // mメンバーが次にやる予定のタスク
				freeTime = minimumWaitTimeCanAssignTask(skill, taskScoreMinMember, nextTask)
			}
			deadline := day + freeTime //この日時までには確実に暇でいる必要がある

			endTime := day + tmpScoreAll[m][t]
			if memberStatus[m] == 1 {
				endTime += calcWaitTime(m)
				// continue //debug用
			}
			// fmt.Printf("#member = %d, freeTime = %d, deadline = %d, endTime = %d\n", m, freeTime, deadline, endTime)

			if deadline+FREE_MARGIN < endTime { //期日までに終わらせられないのでだめ, 上振れ考慮してマージン入れた方が良い
				continue
			}
			// fmt.Printf("#member = %d, endTime = %d\n", m, endTime)
			if endTime < bestEndTime {
				bestEndTime = endTime
				bestMember = m
				bestDeadline = deadline
			}
		}
		if bestMember != -1 && bestEndTime < trueEndTime { //暇人の中から良さそうな人発見
			//次のタスクに確定する
			if memberStatus[bestMember] == 1 {
				continue //その人が暇になるまで待つ
			} else {
				_ = bestDeadline
				// fmt.Printf("#bestMember = %d, bestEndTime = %d, deadline = %d\n", bestMember, bestEndTime, bestDeadline)
				deleteBooking(t)
				memberBookingTask[bestMember] = append([]int{t}, memberBookingTask[bestMember]...)
				taskIsBookedBy[t] = bestMember
			}

			for i := 0; i < len(remainMember); i++ {
				if remainMember[i] == bestMember {
					remainMember = append(remainMember[:i], remainMember[i+1:]...)
					break
				}
			}
		}
	}
	allTimeSearchCalc2 += time.Now().Sub(searchCalc2Start)
}

var memo [1000]int

func minimumWaitTimeCanAssignTask(skill [20][20]int, taskScoreMinMember [1000]int, task int) int { //taskがassign可能になるまでに必要な時間の推定値(各タスクは最も得意な人間が実行するものとする)(メモ化可能)
	if memo[task] != -1 {
		return memo[task]
	}
	ret := 0
	for k := 0; k < len(V[task]); k++ {
		nextT := V[task][k]
		if taskStatus[nextT] != 2 { //完了済みのタスクは無視
			cost := 0
			if taskStatus[nextT] == 0 {
				m := taskScoreMinMember[nextT]
				cost = max(1, scoreTrue(skill[m], nextT)-3) //最も得意な人が実行する想定 上振れも考慮する?
			} else if taskStatus[nextT] == 1 { //実行中タスク
				m := taskIsBookedBy[nextT]
				cost = taskStart[nextT] + max(1, scoreTrue(skill[m], nextT)-3) - day
			}
			ret = max(ret, minimumWaitTimeCanAssignTask(skill, taskScoreMinMember, nextT)+cost)
		}
	}
	memo[task] = ret
	return ret
}

func calcWaitTime(member int) int { //そのメンバーの再アサインが可能になるまで最短でどれぐらいの時間がかかるか(bookingは考慮しない)
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
	return ret - day
}

func deleteBooking(task int) { //taskの予約を削除する
	for i := 0; i < M; i++ {
		foundIdx := -1
		for idx, t := range memberBookingTask[i] {
			if t == task {
				foundIdx = idx
				taskIsBookedBy[t] = -1
				break
			}
		}
		if foundIdx != -1 {
			memberBookingTask[i] = append(memberBookingTask[i][:foundIdx], memberBookingTask[i][foundIdx+1:]...)
			break
		}
	}
}

func findTask(member int) int { //最適なタスクを選定する
	//終了していないタスクの中から最適なタスクにアサインする
	//rankが高い順に処理されることに注意(sortedTasksが既にrank順でソート済み)
	for _, t := range sortedTasks2 {
		if !canAssign(t, true) {
			continue
		}
		return t
	}
	return -1
}

func canAssign(task int, bookingSkip bool) bool {
	if taskStatus[task] != 0 {
		//タスクが終わったか誰かやってる
		return false
	}
	if bookingSkip && taskIsBookedBy[task] != -1 {
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
	var stV int
	var ti int
	var success bool
	var value int
	memberHistory := memberHistory
	l := 0
	taskStart := taskStart
	taskEnd := taskEnd
	taskStatus := taskStatus
	transitionNum := 0

	var st [1000]int

	for _, t := range memberHistory[member] {
		if taskStatus[t] != 2 {
			continue
		}
		for k := 0; k < K; k++ {
			st[t] += stk(now, t, k)
		}
	}

	for {
		targetK = rand.Intn(K)
		targetK2 = targetK
		if rand.Intn(2) == 0 {
			targetK2 = rand.Intn(K)
		}
		add = rand.Intn(2) == 0
		value = rand.Intn(2) + 1
		for _, t := range memberHistory[member] {
			if taskStatus[t] != 2 {
				continue
			}
			st[t] -= stk(now, t, targetK)
			if targetK2 != targetK {
				st[t] -= stk(now, t, targetK2)
			}
		}
		if add {
			now[targetK] = min(sMax[targetK], now[targetK]+value)
			if targetK2 != targetK {
				now[targetK2] = max(psMin[member][targetK2], now[targetK2]-value)
			}
		} else {
			now[targetK] = max(psMin[member][targetK], now[targetK]-value)
			if targetK2 != targetK {
				now[targetK2] = min(sMax[targetK2], now[targetK2]+value)
			}
		}
		for _, t := range memberHistory[member] {
			if taskStatus[t] != 2 {
				continue
			}
			st[t] += stk(now, t, targetK)
			if targetK2 != targetK {
				st[t] += stk(now, t, targetK2)
			}
		}

		success = false

		error = 0
		for _, t := range memberHistory[member] {
			if taskStatus[t] != 2 {
				continue
			}
			//今までに実行した全てのタスクから二乗誤差を算出
			stV = max(1, st[t])
			ti = taskEnd[t] - taskStart[t]
			error += (stV - ti) * (stV - ti)
		}

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
			transitionNum++
		} else { //巻き戻す
			for _, t := range memberHistory[member] {
				if taskStatus[t] != 2 {
					continue
				}
				st[t] -= stk(now, t, targetK)
				if targetK2 != targetK {
					st[t] -= stk(now, t, targetK2)
				}
			}
			if add {
				now[targetK] = max(psMin[member][targetK], now[targetK]-value)
				if targetK2 != targetK {
					now[targetK2] = min(sMax[targetK2], now[targetK2]+value)
				}
			} else {
				now[targetK] = min(sMax[targetK], now[targetK]+value)
				if targetK2 != targetK {
					now[targetK2] = max(psMin[member][targetK2], now[targetK2]-value)
				}
			}
			for _, t := range memberHistory[member] {
				if taskStatus[t] != 2 {
					continue
				}
				st[t] += stk(now, t, targetK)
				if targetK2 != targetK {
					st[t] += stk(now, t, targetK2)
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

	if DEBUG {
		memberHistoryNum := 0
		for _, t := range memberHistory[member] {
			if taskStatus[t] != 2 {
				continue
			}
			memberHistoryNum++
		}
		error := 0
		for k := 0; k < K; k++ {
			error += (sTrue[member][k] - ps[member][k]) * (sTrue[member][k] - ps[member][k])
		}
		bestTrueError := calcError(sTrue[member], member)
		eh := EstimateHistory{bestTrueError: bestTrueError, bestError: bestError, bestError2: calcError2(ps[member], member), memberHistoryNum: memberHistoryNum, transitionNum: transitionNum, trueError: error}
		estimateHistory[member] = append(estimateHistory[member], eh)
	}
}

func stk(skill [20]int, t int, k int) int {
	return max(0, d[t][k]-skill[k])
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
		// if abs(si-ti) <= 3 { //上振れ下振れを考慮
		// 	continue
		// }
		error += (si - ti) * (si - ti)
	}
	return error
}

//こちらは上振れ下振れも考慮
func calcError2(skill [20]int, member int) int {
	error := 0
	for _, t := range memberHistory[member] {
		if taskStatus[t] != 2 {
			continue
		}
		//今までに実行した全てのタスクから二乗誤差を算出
		si := scoreTrue(skill, t)
		ti := taskEnd[t] - taskStart[t]
		if abs(si-ti) <= 3 { //上振れ下振れを考慮
			continue
		}
		error += (si - ti) * (si - ti)
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

func calcRank2(task int, cost int) {
	if cost < rank2[task] {
		//計算済みのrankの方が上の場合無駄なので省略
		return
	}
	rank2[task] = cost

	next := V[task]
	for _, nextT := range next {
		calcRank2(nextT, cost+taskSize[nextT])
	}
}

func calcRank3(skill [20][20]int, taskScoreMinMember [1000]int, task int, cost int) {
	if cost < rank3[task] {
		//計算済みのrankの方が上の場合無駄なので省略
		return
	}
	rank3[task] = cost

	next := V[task]
	for _, nextT := range next {
		if taskStatus[nextT] != 2 { //完了済みのタスクは無視
			score := 0
			if taskStatus[nextT] == 0 {
				m := taskScoreMinMember[nextT]
				score = max(1, scoreTrue(skill[m], nextT)-3) //最も得意な人が実行する想定 上振れも考慮する?
			} else if taskStatus[nextT] == 1 { //実行中タスク
				m := taskIsBookedBy[nextT]
				score = taskStart[nextT] + max(1, scoreTrue(skill[m], nextT)-3) - day
			}
			calcRank3(skill, taskScoreMinMember, nextT, cost+score)
		}
	}
}

func scoreTrue(skill [20]int, task int) int {
	score := 0
	for k := 0; k < K; k++ {
		score += max(0, d[task][k]-skill[k])
	}
	return max(1, score)
}

func abs(a int) int {
	if a < 0 {
		return -a
	}
	return a
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

func writeEstHistory() error {

	for k := 0; k < K; k++ {
		ps[11][k] = 0
	}

	for l := 0; l < 100; l++ {
		estimate(11)
	}

	text := ""
	for i := 0; i < M; i++ {
		text += fmt.Sprintf("member = %d\n", i)
		before := 0
		for idx, eh := range estimateHistory[i] {
			if eh.memberHistoryNum != before {
				text += "\n"
				before++
				if eh.memberHistoryNum != 0 {
					t := memberHistory[i][eh.memberHistoryNum-1]
					text += fmt.Sprintf("t = %d, taskSize = %d, score = %d\n", t, taskSize[t], tTrue[t][i])
				}
			}
			text += fmt.Sprintf("idx = %d, bestTrueError = %d, bestError = %d, bestError2 = %d, trueError = %d, memberHistoryNum = %d, transitionNum = %d\n", idx, eh.bestTrueError, eh.bestError, eh.bestError2, eh.trueError, eh.memberHistoryNum, eh.transitionNum)
		}
		text += "\n"
	}

	file, err := os.Create("./esthistory.txt")
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = file.WriteString(text)
	if err != nil {
		return err
	}

	return nil
}
