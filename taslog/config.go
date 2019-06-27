//   Copyright (C) 2018 ZVChain
//
//   This program is free software: you can redistribute it and/or modify
//   it under the terms of the GNU General Public License as published by
//   the Free Software Foundation, either version 3 of the License, or
//   (at your option) any later version.
//
//   This program is distributed in the hope that it will be useful,
//   but WITHOUT ANY WARRANTY; without even the implied warranty of
//   MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
//   GNU General Public License for more details.
//
//   You should have received a copy of the GNU General Public License
//   along with this program.  If not, see <https://www.gnu.org/licenses/>.

package taslog

const (
	DefaultConfig = `<seelog minlevel="debug">
						<outputs formatid="default">
							<rollingfile type="size" filename="./logs/defaultLOG_INDEX.log" maxsize="100000000" maxrolls="1"/>
						</outputs>
						<formats>
							<format id="default" format="%Date(2006-01-02 15:04:05.000)[%File:%Line] %Msg%n" />
						</formats>
					</seelog>`

	CoreLogConfig = `<seelog minlevel="debug">
						<outputs formatid="default">
							<rollingfile type="size" filename="./logs/coreLOG_INDEX.log" maxsize="200000000" maxrolls="1"/>
						</outputs>
						<formats>
							<format id="default" format="%Date(2006-01-02 15:04:05.000)[%File:%Line]%Msg%n" />
						</formats>
					</seelog>`

	BlockSyncLogConfig = `<seelog minlevel="debug">
						<outputs formatid="default">
							<rollingfile type="size" filename="./logs/block_syncLOG_INDEX.log" maxsize="200000000" maxrolls="1"/>
						</outputs>
						<formats>
							<format id="default" format="%Date(2006-01-02 15:04:05.000)[%File:%Line]%Msg%n" />
						</formats>
					</seelog>`

	GroupSyncLogConfig = `<seelog minlevel="debug">
						<outputs formatid="default">
							<rollingfile type="size" filename="./logs/group_syncLOG_INDEX.log" maxsize="200000000" maxrolls="1"/>
						</outputs>
						<formats>
							<format id="default" format="%Date(2006-01-02 15:04:05.000)[%File:%Line]%Msg%n" />
						</formats>
					</seelog>`

	LockLogConfig = `<seelog minlevel="debug">
						<outputs formatid="default">
							<rollingfile type="size" filename="./logs/lockLOG_INDEX.log" maxsize="100000000" maxrolls="1"/>
						</outputs>
						<formats>
							<format id="default" format="%Date(2006-01-02 15:04:05.000)[%File:%Line]%Msg%n" />
						</formats>
					</seelog>`

	P2PLogConfig = `<seelog minlevel="debug">
						<outputs formatid="default">
							<rollingfile type="size" filename="./logs/p2pLOG_INDEX.log" maxsize="200000000" maxrolls="1"/>
						</outputs>
						<formats>
							<format id="default" format="%Date(2006-01-02 15:04:05.000)[%File:%Line]%Msg%n" />
						</formats>
					</seelog>`

	ConsensusLogConfig = `<seelog minlevel="debug">
						<outputs formatid="default">
							<rollingfile type="size" filename="./logs/consensusLOG_INDEX.log" maxsize="200000000" maxrolls="1"/>
						</outputs>
						<formats>
							<format id="default" format="%Date(2006-01-02 15:04:05.000)[%File:%Line]%Msg%n" />
						</formats>
					</seelog>`

	StdConsensusLogConfig = `<seelog minlevel="debug">
						<outputs formatid="default">
							<rollingfile type="size" filename="./logs/std_consensusLOG_INDEX.log" maxsize="100000000" maxrolls="1"/>
						</outputs>
						<formats>
							<format id="default" format="%Date(2006-01-02 15:04:05.000)[%File:%Line]%Msg%n" />
						</formats>
					</seelog>`
	GroupLogConfig = `<seelog minlevel="debug">
						<outputs formatid="default">
							<rollingfile type="size" filename="./logs/groupLOG_INDEX.log" maxsize="100000000" maxrolls="1"/>
						</outputs>
						<formats>
							<format id="default" format="%Date(2006-01-02 15:04:05.000)[%File:%Line]%Msg%n" />
						</formats>
					</seelog>`
	StatisticsLogConfig = `<seelog minlevel="debug">
						<outputs formatid="default">
							<rollingfile type="size" filename="./logs/statisticsLOG_INDEX.log" maxsize="100000000" maxrolls="1"/>
						</outputs>
						<formats>
							<format id="default" format="%Date(2006-01-02 15:04:05.000)[%File:%Line]%Msg%n" />
						</formats>
					</seelog>`
	VRFDebugLogConfig = `<seelog minlevel="debug">
						<outputs formatid="default">
							<rollingfile type="size" filename="./logs/vrf_debugLOG_INDEX.log" maxsize="100000000" maxrolls="1"/>
						</outputs>
						<formats>
							<format id="default" format="%Date(2006-01-02 15:04:05.000)[%File:%Line]%Msg%n" />
						</formats>
					</seelog>`
	StateMachineLogConfig = `<seelog minlevel="debug">
						<outputs formatid="default">
							<rollingfile type="size" filename="./logs/state_machineLOG_INDEX.log" maxsize="100000000" maxrolls="1"/>
						</outputs>
						<formats>
							<format id="default" format="%Date(2006-01-02 15:04:05.000)[%File:%Line]%Msg%n" />
						</formats>
					</seelog>`

	RewardStatConfig = `<seelog minlevel="debug">
						<outputs formatid="default">
							<rollingfile type="size" filename="./logs/reward_statLOG_INDEX.log" maxsize="100000000" maxrolls="1"/>
						</outputs>
						<formats>
							<format id="default" format="%Date(2006-01-02 15:04:05.000)|%Msg%n" />
						</formats>
					</seelog>`
	MiddlewareLogConfig = `<seelog minlevel="debug">
						<outputs formatid="default">
							<rollingfile type="size" filename="./logs/middlewareLOG_INDEX.log" maxsize="100000000" maxrolls="1"/>
						</outputs>
						<formats>
							<format id="default" format="%Date(2006-01-02 15:04:05.000)[%File:%Line]%Msg%n" />
						</formats>
					</seelog>`
	ForkLogConfig = `<seelog minlevel="debug">
						<outputs formatid="default">
							<rollingfile type="size" filename="./logs/forkLOG_INDEX.log" maxsize="100000000" maxrolls="1"/>
						</outputs>
						<formats>
							<format id="default" format="%Date(2006-01-02 15:04:05.000)[%File:%Line]%Msg%n" />
						</formats>
					</seelog>`
	SlowLogConfig = `<seelog minlevel="debug">
						<outputs formatid="default">
							<rollingfile type="size" filename="./logs/slow_logLOG_INDEX.log" maxsize="100000000" maxrolls="1"/>
						</outputs>
						<formats>
							<format id="default" format="%Date(2006-01-02 15:04:05.000)|%Msg%n" />
						</formats>
					</seelog>`
	TxSyncLogConfig = `<seelog minlevel="debug">
						<outputs formatid="default">
							<rollingfile type="size" filename="./logs/tx_syncLOG_INDEX.log" maxsize="200000000" maxrolls="1"/>
						</outputs>
						<formats>
							<format id="default" format="%Date(2006-01-02 15:04:05.000)[%File:%Line]%Msg%n" />
						</formats>
					</seelog>`
	PerformTraceConfig = `<seelog minlevel="debug">
						<outputs formatid="default">
							<rollingfile type="size" filename="./logs/perform.log" maxsize="200000000" maxrolls="1"/>
						</outputs>
						<formats>
							<format id="default" format="%Msg%n" />
						</formats>
					</seelog>`
)
