package main

import (
	"bufio"
	"fmt"
	"os"
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
	for i := 0; i < R; i++ {
		fmt.Scanf("%d %d", &u, &v)
		u -= 1 //indexを0に揃える
		v -= 1
		V[v] = append(V[v], u) //タスクvはタスクuに依存している
	}

	//デバッグ用, memberの真のスキルを読み込む
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
		var nexta []int
		var nextb []int
		for i := 0; i < M; i++ {
			if memberStatus[i] == 1 {
				continue
			}

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
				bestTask = t
				break
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
		}
	}
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
