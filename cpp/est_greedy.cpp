using namespace std;

#include <time.h>

#include <algorithm>
#include <cstdlib>
#include <fstream>
#include <iostream>
#include <vector>

bool DEBUG = true;
int MIN_ESTIMATE_LEN = 0;
int HC_LOOP_COUNT = 500;
int BOOKING_MARGIN = 4;

int calcNum = 0;

int day = 0;  //現在日時
int N, M, K, R;
int d[1000][20];
vector<int> V[1000];
int taskStatus[1000];
int taskIsBookedBy[1000];
int memberStatus[20];
vector<int> memberHistory[20];
int memberEstimated[20];
vector<int> memberBookingTask[20];
int ps[20][20];
int psMin[20][20];
int sTrue[20][20];
int tTrue[1000][20];
int taskStart[1000];
int taskEnd[1000];
int taskSize[1000];
int rank1[1000];
int rank2[1000];
int sMax[20];
vector<int> sortedTasks;

void calcRank1(int task, int depth);
void calcRank2(int task, int cost);
int size(int task);
bool taskRankComp(int taskA, int taskB);
bool memberHistoryLenComp(int memberA, int memberB);
int findTaskGreedy();
bool canAssign(int task, bool bookingSkip);
int calcError(int skill[20], int member);
int score(int skill[20], int task);
int skillSize(int skill[20]);
void writeScore(int error);

// 山登り法でスキルの推定を行う
void estimate(int member) {
    int bestError = 10000000;
    int bestSkill[20];
    int now[20];
    for (int k = 0; k < K; k++) {
        //初期解は前回の推定結果
        now[k] = ps[member][k];
        bestSkill[k] = ps[member][k];
    }
    bestError = calcError(bestSkill, member);

    int targetK, targetK2;
    bool isAdd;
    int error;
    bool success;
    int l = 0;
    while (true) {
        targetK = rand() % K;
        targetK2 = targetK;
        if (rand() % 3 == 0) {
            // 2箇所同時に変化させてみる
            targetK2 = rand() % K;
        }
        isAdd = (rand() % 2 == 0);
        if (isAdd) {
            now[targetK] = min(sMax[targetK], now[targetK] + 1);
            if (targetK2 != targetK) {
                now[targetK2] = max(psMin[member][targetK2], now[targetK2] - 1);
            }
        } else {
            now[targetK] = max(psMin[member][targetK], now[targetK] - 1);
            if (targetK2 != targetK) {
                now[targetK2] = min(sMax[targetK2], now[targetK2] + 1);
            }
        }

        success = false;
        error = calcError(now, member);
        if (error == bestError) {
            //エラーが同じ場合はskillがより小規模なもの
            success = (skillSize(now) < skillSize(bestSkill));
        } else if (error < bestError) {
            success = true;
        }

        if (success) {
            bestError = error;
            for (int k = 0; k < K; k++) {
                bestSkill[k] = now[k];
            }
        } else {
            //巻き戻す
            if (isAdd) {
                now[targetK] = max(psMin[member][targetK], now[targetK] - 1);
                if (targetK2 != targetK) {
                    now[targetK2] = min(sMax[targetK2], now[targetK2] + 1);
                }
            } else {
                now[targetK] = min(sMax[targetK], now[targetK] + 1);
                if (targetK2 != targetK) {
                    now[targetK2] =
                        max(psMin[member][targetK2], now[targetK2] - 1);
                }
            }
        }

        l++;
        if (l == HC_LOOP_COUNT) {
            //最終的にはタイマーにするべき
            break;
        }
    }

    for (int k = 0; k < K; k++) {
        ps[member][k] = bestSkill[k];
    }
}

int main() {
    clock_t startTime = clock();
    cout << "#test" << endl;
    cin >> N >> M >> K >> R;
    cout << "#N:" << N << endl;

    for (int j = 0; j < K; j++) {
        sMax[j] = 0;
    }

    for (int i = 0; i < N; i++) {
        for (int j = 0; j < K; j++) {
            cin >> d[i][j];
            sMax[j] = max(sMax[j], d[i][j]);
        }
        taskSize[i] = size(i);
        taskIsBookedBy[i] = -1;
        taskStatus[i] = 0;
    }

    for (int m = 0; m < M; m++) {
        memberStatus[m] = 0;
        memberEstimated[m] = -1;
        for (int k = 0; k < K; k++) {
            ps[m][k] = 0;
            psMin[m][k] = 0;
        }
    }

    int u;
    int v;
    for (int i = 0; i < R; i++) {
        cin >> u >> v;
        u -= 1;
        v -= 1;
        V[v].push_back(u);
    }

    if (DEBUG) {
        for (int i = 0; i < M; i++) {
            for (int k = 0; k < K; k++) {
                cin >> sTrue[i][k];
                if (sMax[k] < sTrue[i][k]) {
                    sMax[k] = sTrue[i][k];
                }
            }
        }
        for (int t = 0; t < N; t++) {
            for (int i = 0; i < M; i++) {
                cin >> tTrue[t][i];
            }
        }
    }

    // rank計算
    for (int t = 0; t < N; t++) {
        rank1[t] = -1;
        rank2[t] = 0;
    }
    for (int t = 0; t < N; t++) {
        calcRank1(t, 0);
        calcRank2(t, taskSize[t]);
    }

    // rankが大きい順にtaskを並べておく(rankが大きい物から処理しないと後半ボトルネックになる)
    for (int t = 0; t < N; t++) {
        sortedTasks.push_back(t);
    }
    sort(sortedTasks.begin(), sortedTasks.end(), taskRankComp);
    for (int i = 0; i < N; i++) {  //ソート結果を表示
        int t = sortedTasks[i];
        cout << "# " << t << " size = " << taskSize[t]
             << " rank1 = " << rank1[t] << " rank2 = " << rank2[t] << endl;
    }

    int n, f;
    while (true) {
        cout << "#day = " << day << endl;

        vector<int> nexta;
        vector<int> nextb;

        //処理したタスクが多い順にmeberをソートする もう不要?
        vector<int> sortedMembers;
        for (int i = 0; i < M; i++) {
            sortedMembers.push_back(i);
        }
        sort(sortedMembers.begin(), sortedMembers.end(), memberHistoryLenComp);
        for (int i = 0; i < M; i++) {
            int m = sortedMembers[i];
            cout << "# member = " << m << ", memberStatus = " << memberStatus[m]
                 << ", len = " << memberHistory[m].size() << endl;
        }

        int estimatedMemberNum = 0;
        for (size_t i = 0; i < sortedMembers.size(); i++) {
            int m = sortedMembers[i];
            if (memberEstimated[m] == 1) {
                estimatedMemberNum++;
            }
        }

        //貪欲法
        for (size_t idx = 0; idx < sortedMembers.size(); idx++) {
            int m = sortedMembers[idx];
            if (memberStatus[m] == 1) {
                //作業中のメンバーはskip
                continue;
            }
            if (memberBookingTask[m].size() != 0) {
                //タスク予約中なのでskip
                continue;
            }

            int bestTask = findTaskGreedy();
            if (bestTask == -1) {
                continue;
            }
            memberBookingTask[m].push_back(bestTask);
            taskIsBookedBy[bestTask] = m;
        }

        //アサイン処理
        for (int m = 0; m < M; m++) {
            if (memberStatus[m] == 1) {
                continue;
            }
            if (memberBookingTask[m].size() == 0) {
                continue;
            }

            int t =
                memberBookingTask[m][0];  //予約テーブルの手前からアサインする
            if (!canAssign(t, false)) {
                continue;
            }

            memberBookingTask[m].erase(
                memberBookingTask[m].begin());  //先頭をpop

            nexta.push_back(m);
            nextb.push_back(t);
            taskStatus[t] = 1;
            memberStatus[m] = 1;
            memberHistory[m].push_back(t);
            taskStart[t] = day;
        }

        //結果を出力
        cout << nexta.size();
        for (size_t i = 0; i < nexta.size(); i++) {
            int a = nexta[i] + 1;
            int b = nextb[i] + 1;
            cout << " " << a << " " << b;
        }
        cout << endl;

        //スキルの予測値を出力
        int error = 0;
        if (DEBUG) {
            for (int i = 0; i < M; i++) {
                cout << "#s " << (i + 1);
                for (int k = 0; k < K; k++) {
                    cout << " " << ps[i][k];
                    if (memberEstimated[i] == 1) {
                        error +=
                            (ps[i][k] - sTrue[i][k]) * (ps[i][k] - sTrue[i][k]);
                    }
                }
                cout << endl;
            }
            if (estimatedMemberNum != 0) {
                error /= estimatedMemberNum;
                cout << "# estimate error = " << error << endl;
            }
        }

        //処理時間を計測
        if (DEBUG) {
            clock_t now = clock();
            double time =
                static_cast<double>(now - startTime) / CLOCKS_PER_SEC * 1000.0;
            cout << "#allTime = " << time << "ms" << endl;
            cout << "#calcNum = " << calcNum << endl;
        }

        day++;

        cin >> n;
        if (n == -1) {
            if (DEBUG) {
                writeScore(error);
            }
            break;
        }
        for (int i = 0; i < n; i++) {
            cin >> f;
            f -= 1;  // indexを0に揃える
            memberStatus[f] = 0;
            int t = memberHistory[f][memberHistory[f].size() - 1];
            taskStatus[t] = 2;  // taskをdoneに
            taskEnd[t] = day;

            //学習データが溜まったらパラメータを推定する
            if (memberHistory[f].size() > MIN_ESTIMATE_LEN) {
                estimate(f);
                memberEstimated[f] = 1;
            }

            //パラメータの下限が確定(下振れを考慮)
            int actDay = taskEnd[t] - taskStart[t];
            for (int k = 0; k < K; k++) {
                psMin[f][k] = max(psMin[f][k], d[t][k] - actDay - 3);
                ps[f][k] = max(psMin[f][k], ps[f][k]);
            }
        }
    }

    return 0;
}

int findTaskGreedy() {
    //終了していないタスクの中から最もrankが高いものを貪欲に採用
    for (size_t i = 0; i < sortedTasks.size(); i++) {
        int t = sortedTasks[i];
        if (!canAssign(t, true)) {
            continue;
        }
        return t;
    }
    return -1;
}

//今までに実行したすべてのタスクから二乗誤差の総和を算出
int calcError(int skill[20], int member) {
    int error = 0;
    for (size_t i = 0; i < memberHistory[member].size(); i++) {
        int t = memberHistory[member][i];
        if (taskStatus[t] != 2) {
            //完了していないタスクは計算に含めない
            continue;
        }
        int si = score(skill, t);
        int ti = taskEnd[t] - taskStart[t];
        error += (si - ti) * (si - ti);
    }
    return error;
}
int score(int skill[20], int task) {
    int score = 0;
    for (int k = 0; k < K; k++) {
        calcNum++;
        score += max(0, d[task][k] - skill[k]);
    }
    return max(1, score);
}
int skillSize(int skill[20]) {
    int sum = 0;
    for (int k = 0; k < K; k++) {
        sum += skill[k];
    }
    return sum;
}

bool canAssign(int task, bool bookingSkip) {
    if (taskStatus[task] != 0) {
        //タスクが終わった or 誰かやってる
        return false;
    }
    if (bookingSkip && taskIsBookedBy[task] != -1) {
        //タスクが予約済み
        return false;
    }

    //依存するタスクが終わっていない場合は実行不可能
    bool canAssign = true;
    for (int i = 0; i < V[task].size(); i++) {
        int u = V[task][i];
        if (taskStatus[u] != 2) {
            //終わっていない
            canAssign = false;
            break;
        }
    }
    return canAssign;
}

void calcRank1(int task, int depth) {
    if (depth < rank1[task]) {
        //計算済みのrankの方が上の場合無駄なので省略
        return;
    }
    rank1[task] = depth;

    for (int i = 0; i < V[task].size(); i++) {
        int nextT = V[task][i];
        calcRank1(nextT, depth + 1);
    }
}

void calcRank2(int task, int cost) {
    if (cost < rank2[task]) {
        //計算済みのrankの方が上の場合無駄なので省略
        return;
    }
    rank2[task] = cost;

    for (int i = 0; i < V[task].size(); i++) {
        int nextT = V[task][i];
        calcRank2(nextT, cost + taskSize[nextT]);
    }
}

int size(int task) {
    int sum = 0;
    for (int k = 0; k < K; k++) {
        sum += d[task][k];
    }
    return sum;
}

bool taskRankComp(int taskA, int taskB) {
    if (rank1[taskA] == rank1[taskB]) {
        return rank2[taskA] > rank2[taskB];
    }
    return rank1[taskA] > rank1[taskB];
}

bool memberHistoryLenComp(int memberA, int memberB) {
    return memberHistory[memberA].size() > memberHistory[memberB].size();
}

void writeScore(int error) {
    std::ofstream writing_file;
    std::string filename = "./estscore.txt";
    writing_file.open(filename, std::ios::out);
    writing_file << error << std::endl;
    writing_file.close();
}
