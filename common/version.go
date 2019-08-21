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

package common

//const GtasVersion = "0.6.2"
//const GtasVersion = "0.6.3"	//add log committing to mysql
//const GtasVersion = "0.7.0"	//optimize group creating process, and make the number of group be flexible
//const GtasVersion = "0.7.1"	//bugFix for multi new groups created at one block high, make pong reference to block high
//const GtasVersion = "0.7.2"	//add asking share piece function when members can't collect all share pieces in group creating process
//const GtasVersion = "0.7.3"	//optimize transaction pool's parameters and p2p sequence
//const GtasVersion = "0.7.4"	//modify p2p parameter to 1024
//const GtasVersion = "0.7.5"	//optimize group signature process: ignore the block with lower qn
//const GtasVersion = "0.7.7"	//bugFix for message relay in P2P network
//const GtasVersion = "0.8.0"		//optimize operation of chain
//const GtasVersion = "0.8.1"		//bugFix: make sure that the current block time will be after the previous.
//const GtasVersion = "0.8.2"		//time sync and adjustment
//const GtasVersion = "0.9.0" //add pledge agency function, verifiers ignore the correctness of transactions in block
//const GtasVersion = "0.9.3" // add comments
//const GtasVersion = "0.9.10" // add comments
//const GtasVersion = "0.9.11" // fix group member check

const GtasVersion = "0.9.13"

const ConsensusVersion = 1

const ChainDataVersion = 12

const ProtocolVersion = 1
