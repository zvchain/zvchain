//   Copyright (C) 2019 ZVChain
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

package update

type UpdateInfo struct {
	PackageUrl  string   `json:"package_url"`
	PackageMd5  string   `json:"package_md5"`
	PackageSign string   `json:"package_sign"`
	FileList    []string `json:"file_list"`
}

type Notice struct {
	Version         string      `json:"version"`
	NotifyGap       string      `json:"notify_gap"`
	EffectiveHeight string      `json:"effective_height"`
	Required        string      `json:"required"`
	NoticeContent   string      `json:"notice_content"`
	WhiteList       []string    `json:"white_list"`
	UpdateInfos     *UpdateInfo `json:"update_info"`
}

type Result struct {
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

type ErrorResult struct {
	Message string `json:"message"`
	Code    int    `json:"code"`
}
