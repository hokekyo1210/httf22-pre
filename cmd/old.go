package main

import (
	"bufio"
	"fmt"
	"math/rand"
	"os"
	"sort"
)

var (
	N             int
	M             int
	K             int
	R             int
	u, v          int
	d             [1000][20]int
	V             [1000][]int //依存関係を管理
	taskStatus    [1000]int   //タスクステータス管理, 0:not yet, 1:working, 2:done
	memberStatus  [20]int     //メンバーステータス管理, 0:free, 1:working
	memberHistory [20][]int   //タスク実行履歴
	ps            [20][20]int //メンバーのステータスの推定値
	sTrue         [20][20]int //メンバーのステータス(本物)
	taskStart     [1000]int   //タスクを開始した時刻
	taskEnd       [1000]int   //タスクを終了した時刻
	taskSize      [1000]int   //タスクの大きさ
	sMax          int         //sの取りうる上限
)

func main() {
	fmt.Scanf("%d %d %d %d", &N, &M, &K, &R)
	for i := 0; i < N; i++ {
		for j := 0; j < K; j++ {
			fmt.Scanf("%d", &d[i][j])
			sMax = max(sMax, d[i][j])
		}
		taskSize[i] = size(i)
	}
	//乱数で初期解生成
	// for i := 0; i < M; i++ {
	// 	for k := 0; k < K; k++ {
	// 		ps[i][k] = rand.Intn(50)
	// 	}
	// }
	for i := 0; i < R; i++ {
		fmt.Scanf("%d %d", &u, &v)
		u -= 1 //indexを0に揃える
		v -= 1
		V[v] = append(V[v], u) //タスクvはタスクuに依存している
	}

	//デバッグ用, memberのスキルを読み込む
	for i := 0; i < M; i++ {
		for k := 0; k < K; k++ {
			fmt.Scanf("%d", &sTrue[i][k])
		}
	}

	var wtr = bufio.NewWriter(os.Stdout)
	var n int
	var f int
	day := 0
	for {
		// for t := 0; t < N; t++ {
		// 	fmt.Printf("task %d is %d depends on [", t+1, taskStatus[t])
		// 	for k := 0; k < len(V[t]); k++ {
		// 		fmt.Printf("%d ", V[t][k]+1)
		// 	}
		// 	fmt.Printf("]\n")
		// }

		//処理したタスクが少ない順にソートする
		var sortedMembers []int
		for i := 0; i < M; i++ {
			sortedMembers = append(sortedMembers, i)
		}
		sort.Slice(sortedMembers, func(i, j int) bool {
			return len(memberHistory[sortedMembers[i]]) > len(memberHistory[sortedMembers[j]])
		})

		var nexta []int
		var nextb []int
		for _, i := range sortedMembers {
			if memberStatus[i] == 1 {
				continue
			}
			if day >= 100 { //ある程度情報が集まるまでは推定しない
				updatePredictSkill(i)
			}

			bestScore := 10000000000
			bestTask := -1
			//終了していないタスクの中から最適なタスクにアサインする
			for t := 0; t < N; t++ {
				if taskStatus[t] != 0 {
					//タスクが終わったか誰かやってる
					continue
				}

				//依存するタスクがあって、そちらが終わってない場合は無理
				canAssign := true
				for k := 0; k < len(V[t]); k++ {
					if taskStatus[V[t][k]] != 2 {
						canAssign = false
					}
				}
				if !canAssign {
					continue
				}

				score := scoreTrue(ps[i], t)
				if score <= bestScore {
					if score == bestScore {
						if taskSize[bestTask] < taskSize[t] {
							bestTask = t
						}
					} else {
						bestTask = t
						bestScore = score
					}
				}
			}
			if bestTask == -1 {
				continue
			}
			nexta = append(nexta, i)
			nextb = append(nextb, bestTask)
			taskStatus[bestTask] = 1
			memberStatus[i] = 1
			memberHistory[i] = append(memberHistory[i], bestTask)
			taskStart[bestTask] = day
		}

		fmt.Fprintf(wtr, "%d", len(nexta))
		for i := 0; i < len(nexta); i++ {
			fmt.Fprintf(wtr, " %d %d", nexta[i]+1, nextb[i]+1) //+1しておかないとインデックスがずれる
		}
		fmt.Fprintf(wtr, "\n")

		for i := 0; i < len(nexta); i++ {
			m := nexta[i]
			fmt.Fprintf(wtr, "#s %d", m+1) //予測値を出力
			for k := 0; k < K; k++ {
				fmt.Fprintf(wtr, " %d", ps[m][k])
			}
			fmt.Fprintf(wtr, "\n")
		}

		err := wtr.Flush() //flushしないとだめ
		if err != nil {
			fmt.Printf("error %s\n", err.Error())
			os.Exit(1)
		}

		day++

		fmt.Scanf("%d", &n)
		if n == -1 {
			break
		}
		for i := 0; i < n; i++ {
			fmt.Scanf("%d", &f)
			f -= 1 //indexを0に揃える
			memberStatus[f] = 0
			t := memberHistory[f][len(memberHistory[f])-1]
			taskStatus[t] = 2 //taskをdoneに
			taskEnd[t] = day
			fmt.Printf("# diff %d %d\n", scoreTrue(ps[f], t), taskEnd[t]-taskStart[t])
		}

		errorAll := 0
		for i := 0; i < M; i++ {
			error := 0
			for j := 0; j < len(memberHistory[i]); j++ {
				t := memberHistory[i][j]
				a1 := scoreTrue(sTrue[i], t)
				a2 := scoreTrue(ps[i], t)
				error += (a1 - a2) * (a1 - a2)
			}
			// for k := 0; k < K; k++ {
			// 	error += (ps[i][k] - sTrue[i][k]) * (ps[i][k] - sTrue[i][k])
			// }
			fmt.Printf("# error(%d, n=%d) = %d\n", i, len(memberHistory[i]), error)
			errorAll += error
		}
		fmt.Printf("# errorAll = %d\n", errorAll)
	}
}

func updatePredictSkill(member int) {
	bestError := 10000000000
	var bestSkill [20]int
	var tmp [20]int
	for k := 0; k < K; k++ {
		tmp[k] = ps[member][k] //最初は前回の結果を使う
	}
	update := 0
	fmt.Printf("# n=%d", len(memberHistory[member]))
	for l := 0; l < 1000; l++ { //とりあえず100回試行
		error := 0
		for i := 0; i < len(memberHistory[member]); i++ {
			//今までに実行した全てのタスクから二乗誤差を算出
			t := memberHistory[member][i]
			si := max(1, scoreTrue(tmp, t))
			ti := taskEnd[t] - taskStart[t]
			// fmt.Printf("#ti : %d\n", ti)
			error += (si - ti) * (si - ti)
		}
		if error < bestError {
			bestError = error
			for k := 0; k < K; k++ {
				bestSkill[k] = tmp[k]
			}
			fmt.Printf(" %d", bestError)
			update++
		}

		//乱数で次のsを生成
		for k := 0; k < K; k++ {
			tmp[k] = rand.Intn(sMax)
		}
	}
	fmt.Printf("\n")
	fmt.Printf("#update %d\n", update)

	//1パラメータずつ舐める
	// for k := 0; k < K; k++ {
	// 	tmp[k] = bestSkill[k]
	// }
	// fmt.Printf("# n=%d", len(memberHistory[member]))
	// for l := 0; l < 1; l++ {
	// 	for k := 0; k < K; k++ {
	// 		beforeV := tmp[k]
	// 		bestV := -1
	// 		for v := 0; v < sMax; v++ {
	// 			tmp[k] = v
	// 			error := calcError(tmp, member)
	// 			if error < bestError {
	// 				bestError = error
	// 				bestV = v
	// 				fmt.Printf(" %d", bestError)
	// 			}
	// 		}
	// 		if bestV != -1 {
	// 			tmp[k] = bestV
	// 		} else {
	// 			tmp[k] = beforeV
	// 		}
	// 	}
	// }
	// fmt.Printf("\n")

	// for k := 0; k < K; k++ {
	// 	bestSkill[k] = tmp[k]
	// }

	// fmt.Printf("#")
	for k := 0; k < K; k++ {
		ps[member][k] = bestSkill[k]
		// fmt.Printf(" %d", ps[member][k])
	}
	// fmt.Printf("\n")
}

func calcError(skill [20]int, member int) int {
	error := 0
	for i := 0; i < len(memberHistory[member]); i++ {
		//今までに実行した全てのタスクから二乗誤差を算出
		t := memberHistory[member][i]
		si := max(1, scoreTrue(skill, t))
		ti := taskEnd[t] - taskStart[t]
		error += (si - ti) * (si - ti)
	}
	return error
}

func score(skill [20]int, task int) int {
	score := 0
	for k := 0; k < K; k++ {
		score += d[task][k] - skill[k]
	}
	if score < 1 {
		score = 1
	}
	return score
}

func scoreTrue(skill [20]int, task int) int {
	score := 0
	for k := 0; k < K; k++ {
		score += max(0, d[task][k]-skill[k])
	}
	return max(1, score)
}

func scoreOriginal(skill [20]int, task int) int {
	score := 0
	for k := 0; k < K; k++ {
		score += (d[task][k] - skill[k]) * (d[task][k] - skill[k])
	}
	return score
}

func scoreK(skill [20]int, task int, k int) int {
	score := 0
	score += d[task][k] - skill[k]
	if score < 1 {
		score = 1
	}
	return score
}

func max(a int, b int) int {
	if a < b {
		return b
	}
	return a
}

func size(task int) int {
	sum := 0
	for _, v := range d[task] {
		sum += v
	}
	return sum
}
