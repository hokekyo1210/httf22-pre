using namespace std;

#include <time.h>

#include <algorithm>
#include <cstdlib>
#include <fstream>
#include <iostream>
#include <vector>

bool DEBUG = true;
int MIN_ESTIMATE_LEN = 0;
int HC_LOOP_COUNT = 100;
int BOOKING_MARGIN = 4;

int calcNum = 0;
int estimateLoopNum = 0;
int estimateLoopSuccessNum = 0;

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
int memo[1000];

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
int calcWaitTime(int member);
int minimumWaitTimeCanAssignTask(int skill[20][20],
                                 int taskScoreMinMember[1000], int task);
void deleteBooking(int task);

// 推定スキルを利用して最適な割当を探索する
void bestAssign() {
    int taskScoreMin[1000];
    int taskScoreMinMember[1000];
    int scoreAll[20];
    int scoreMemo[20][1000];
    vector<pair<int, int> > membersScoreRanking;

    // 初期化
    for (int m = 0; m < M; m++) {
        scoreAll[m] = 0;
    }
    for (int t = 0; t < N; t++) {
        taskScoreMin[t] = 100000000;
        taskScoreMinMember[t] = -1;
    }

    // 全メンバーの未実行全タスクを仮に実行した場合のスコアをすべて計算する
    // タスク毎に最も低いスコアを記録しておく
    int s;
    for (int m = 0; m < M; m++) {
        for (int t = 0; t < N; t++) {  //これランク順手前100個ぐらいで良いかも
            if (taskStatus[t] != 0) {
                //実行済み, 実行中のタスクは計算しない
                continue;
            }
            scoreMemo[m][t] = score(ps[m], t);
            taskScoreMin[t] = min(taskScoreMin[t], scoreMemo[m][t]);
            scoreAll[m] += scoreMemo[m][t];
        }
    }

    // memberをscoreALL大きい順に並べる
    for (int m = 0; m < M; m++) {
        membersScoreRanking.push_back(make_pair(-scoreAll[m], m));
    }
    sort(membersScoreRanking.begin(), membersScoreRanking.end());

    for (size_t i = 0; i < membersScoreRanking.size(); i++) {
        int s = -membersScoreRanking[i].first;
        int m = membersScoreRanking[i].second;
        cout << "# member = " << m << ", s = " << s << endl;
        for (int t = 0; t < N; t++) {
            if (taskStatus[t] != 0) {
                //実行済み, 実行中のタスクは計算しない
                continue;
            }
            if (taskScoreMinMember[t] == -1 &&
                scoreMemo[m][t] == taskScoreMin[t]) {
                taskScoreMinMember[t] = m;
            }
        }
    }

    for (int m = 0; m < M; m++) {
        memberBookingTask[m] = vector<int>();
    }
    // taskScoreMinMemberをベースにどんどんassignしていく
    for (size_t i = 0; i < sortedTasks.size(); i++) {
        int t = sortedTasks[i];
        if (taskStatus[t] != 0) {
            //実行済み, 実行中のタスクは計算しない
            continue;
        }
        int m =
            taskScoreMinMember[t];  //注意, tを絞り込んだ場合ここが-1になるかも
        memberBookingTask[m].push_back(t);
        taskIsBookedBy[t] = m;
    }

    for (int m = 0; m < M; m++) {
        cout << "# member = " << m << ", bookingTasks = [";
        for (size_t i = 0; i < memberBookingTask[m].size(); i++) {
            cout << memberBookingTask[m][i] << " ";
        }
        cout << "]" << endl;
    }

    // 次のタスクまでの間が十分長い人の中から、最も早くタスクを終えられる人を探してassign
    bool remainMemberFlag[20];
    int remainMemberNum = 0;
    for (int m = 0; m < M; m++) {
        remainMemberFlag[m] = false;
        if (memberStatus[m] == 1) {
            // 作業中のメンバーは対象外
            continue;
        }
        remainMemberFlag[m] = true;
        remainMemberNum++;
    }
    for (int t = 0; t < N; t++) {
        // DP用のメモテーブルを初期化
        memo[t] = -1;
    }
    for (size_t i = 0; i < sortedTasks.size(); i++) {
        int t = sortedTasks[i];
        if (remainMemberNum == 0) {  //全員assignされたら終わる
            break;
        }
        if (!canAssign(t,
                       false)) {  //今すぐに実行可能なタスクのみを探索対象にする
            continue;
        }
        int memberIsBooking = taskIsBookedBy[t];  //タスクを予約している人
        if (memberStatus[memberIsBooking] == 0 &&
            memberBookingTask[memberIsBooking][0] == t) {
            // 今日中に実行予定のメンバーがいるのでスキップ
            // 横取りしたほうが良い可能性もある
            continue;
        }
        int trueEndTime =
            day + calcWaitTime(memberIsBooking);  //本来このタスクが終わる時間
        for (size_t j = 0; j < memberBookingTask[memberIsBooking].size(); j++) {
            int bookedT = memberBookingTask[memberIsBooking][j];
            trueEndTime +=
                score(ps[memberIsBooking], bookedT);  //平均的なケースを想定
            if (bookedT == t) {
                break;
            }
        }

        int bestEndTime = 100000000;
        int bestMember = -1;
        for (int m = 0; m < M; m++) {
            if (memberIsBooking == m) {
                continue;
            }
            if (memberBookingTask[m].size() != 0) {
                int nextT = memberBookingTask[m][0];
                if (canAssign(nextT, false)) {
                    // 今すぐに実行可能なタスクを抱えているので考慮不要
                    continue;
                }
            }

            int freeTime = 100000000;
            if (memberBookingTask[m].size() != 0) {
                int nextTask =
                    memberBookingTask[m][0];  // mメンバーが次にやるタスク
                freeTime = minimumWaitTimeCanAssignTask(ps, taskScoreMinMember,
                                                        nextTask);
            }
            int deadline =
                day + freeTime;  //この日時までに確実に暇でいる必要がある
            int endTime = day + score(ps[m], t);
            if (memberStatus[m] == 1) {
                endTime += calcWaitTime(m);
            }
            if (deadline + BOOKING_MARGIN < endTime) {
                //期日までに終わらせられないのでだめ
                continue;
            }
            if (trueEndTime <= endTime) {
                //元より悪化するのでだめ
                continue;
            }

            if (endTime < bestEndTime) {
                bestEndTime = endTime;
                bestMember = m;
            }
        }
        if (bestMember != -1) {
            //良さそうな人発見, タスクを横取りする
            if (memberStatus[bestMember] == 1) {
                //仕事中なら待つ
                continue;
            } else if (remainMemberFlag[bestMember]) {
                cout << "# bestMember = " << bestMember
                     << ", bestEndTime = " << bestEndTime << endl;
                remainMemberFlag[bestMember] = false;
                remainMemberNum--;
                deleteBooking(t);
                memberBookingTask[bestMember].insert(
                    memberBookingTask[bestMember].begin(), t);
                taskIsBookedBy[t] = bestMember;
            }
        }
    }
}

int minimumWaitTimeCanAssignTask(int skill[20][20],
                                 int taskScoreMinMember[1000], int task) {
    if (memo[task] != -1) {
        return memo[task];
    }
    int ret = 0;
    for (size_t i = 0; i < V[task].size(); i++) {
        int nextT = V[task][i];
        if (taskStatus[nextT] != 2) {  //完了済みのタスクは無視
            int cost = 0;
            if (taskStatus[nextT] == 0) {
                int m = taskScoreMinMember[nextT];
                cost = score(
                    skill[m],
                    nextT);  //最も得意な人が実行する想定 上振れも考慮する?
            } else if (taskStatus[nextT] == 1) {  //実行中タスク
                int m = taskIsBookedBy[nextT];
                cost = taskStart[nextT] + score(skill[m], nextT) - day;
            }
            ret = max(ret, minimumWaitTimeCanAssignTask(
                               skill, taskScoreMinMember, nextT) +
                               cost);
        }
    }
    memo[task] = ret;
    return ret;
}

//そのメンバーの再アサインが可能になるまで最短でどれぐらいの時間がかかるか(bookingは考慮しない)
int calcWaitTime(int member) {
    int ret = day;  //いつ終わるか
    if (memberStatus[member] == 1) {
        int working = memberHistory[member][memberHistory[member].size() - 1];
        int tmp = score(ps[member], working);
        int endTime;
        if (tmp == 1) {
            endTime = taskStart[working] + tmp;
        } else {
            endTime = taskStart[working] + tmp + 3;  //上振れも考慮する?
        }
        ret = endTime;
    }
    return ret - day;
}

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
        estimateLoopNum++;
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
            estimateLoopSuccessNum++;
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
        int freeMemberNum = 0;
        for (size_t i = 0; i < sortedMembers.size(); i++) {
            int m = sortedMembers[i];
            if (memberEstimated[m] == 1) {
                estimatedMemberNum++;
            }
            if (memberStatus[m] == 0) {
                freeMemberNum++;
            }
        }

        // 推定スキルを利用して最適な割当を探索する
        if (estimatedMemberNum == M && freeMemberNum != 0) {
            //この条件は再検討する
            bestAssign();
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
            cout << "#estimateLoopNum = " << estimateLoopNum << endl;
            cout << "#estimateLoopSuccessNum = " << estimateLoopSuccessNum
                 << endl;
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

void deleteBooking(int targetTask) {  // taskの予約を削除する
    int m = taskIsBookedBy[targetTask];
    for (size_t i = 0; i < memberBookingTask[m].size(); i++) {
        int t = memberBookingTask[m][i];
        if (t != targetTask) {
            continue;
        }
        memberBookingTask[m].erase(memberBookingTask[m].begin() + i);
        break;
    }
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
