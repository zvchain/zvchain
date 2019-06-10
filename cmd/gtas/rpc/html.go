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

package rpc

const HTMLTEM = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <title>Title</title>
    <link rel="stylesheet" href="http://www.ysh0566.top/js/layui/css/layui.css">
    <style type="text/css">
        .wallet_tr {word-wrap:break-word; word-break:break-all;}
    </style>
</head>
<body>
<div style="margin: 20px 5%" >
    <div>
        <label class="layui-label" id="host"></label>
        <span class="layui-badge-dot" style="position: relative; top: -2px; left: -2px" id="online_flag"></span>
        <button class="layui-btn layui-btn-xs" id="change_host">更换host</button>
    </div>
    <div class="layui-tab" lay-filter="demo" style="margin: 20px 0">
        <ul class="layui-tab-title">
            <li class="layui-this">Dashboard</li>
            <li>新建账户</li>
            <li>转账演示</li>
            <li>查询余额</li>
            <li>投票</li>
            <li>查询块信息</li>
            <li>查询组信息</li>
            <li>查询工作组</li>
            <li>共识验证</li>
        </ul>
        <div class="layui-tab-content">
            <div class="layui-tab-item  layui-show">
                <div class="layui-row layui-col-space10 layui-bg-green">
                    <div class="layui-col-md4">
                    </div>
                    <div class="layui-col-md2" style="font-size: 20px">
                        <span>当前块高：</span><span id="block_height">0</span>
                    </div>
                    <div class="layui-col-md2" style="font-size: 20px">
                        <span>当前组高：</span><span id="group_height">0</span>
                    </div>
                    <div class="layui-col-md2" style="font-size: 20px">
                        <span>工作组数量：</span><span id="work_group_num">0</span>
                    </div>
                    <div class="layui-col-md4">
                    </div>
                </div>
                <hr/>
                <div class="layui-row layui-col-space10">
                    <div class="layui-col-md6">
                        <label>已连接节点</label>
                        <table class="layui-table">
                            <colgroup>
                                <col width="50%">
                                <col width="30%">
                                <col>
                            </colgroup>
                            <thead>
                            <tr>
                                <th>id</th>
                                <th>ip地址</th>
                                <th>端口</th>
                            </tr>
                            </thead>
                            <tbody id="nodes_table">

                            </tbody>
                        </table>
                    </div>
                    <div class="layui-col-md6">
                        <label>缓冲区</label>
                        <table class="layui-table">
                            <colgroup>
                                <col width="33%">
                                <col width="33%">
                                <col>
                            </colgroup>
                            <thead>
                            <tr>
                                <th>source</th>
                                <th>target</th>
                                <th>value</th>
                            </tr>
                            </thead>
                            <tbody id="trans_table">

                            </tbody>
                        </table>
                    </div>
                </div>
            </div>
            <div class="layui-tab-item">
                <button class="layui-btn" id="create_btn">创建账户</button>
                <table class="layui-table" style="table-layout: fixed">
                    <colgroup>
                        <col width="65%">
                        <col width="20%">
                        <col>
                    </colgroup>
                    <thead>
                    <tr>
                        <th>私钥</th>
                        <th>钱包地址</th>
                        <th>操作</th>
                    </tr>
                    </thead>
                    <tbody id="create_chart">

                    </tbody>
                </table>
            </div>
            <div class="layui-tab-item">
                <form class="layui-form" action="">
                    <div class="layui-form-item">
                        <label class="layui-form-label">from</label>
                        <div class="layui-input-block">
                            <input type="text" name="from"   autocomplete="off" class="layui-input"
                                   placeholder="请输入发送方地址(len=40)">
                        </div>
                    </div>
                    <div class="layui-form-item">
                        <label class="layui-form-label">to</label>
                        <div class="layui-input-block">
                            <input type="text" name="to"   autocomplete="off" class="layui-input"
                                   placeholder="请输入接收方地址(len=40)">
                        </div>
                    </div>
                    <div class="layui-form-item">
                        <label class="layui-form-label">amount</label>
                        <div class="layui-input-block">
                            <input type="text" name="amount"  autocomplete="off" class="layui-input"
                                   placeholder="请输入金额(正整数)">
                        </div>
                    </div>
                    <div class="layui-form-item">
                        <label class="layui-form-label">code</label>
                        <div class="layui-input-block">
                            <input type="text" name="code"  autocomplete="off" class="layui-input"
                                   placeholder="code">
                        </div>
                    </div>
                    <div class="layui-form-item">
                        <div class="layui-input-block">
                            <button class="layui-btn" lay-submit lay-filter="t_form">立即提交</button>
                            <button type="reset" class="layui-btn layui-btn-primary">重置</button>
                        </div>
                    </div>
                </form>
                <hr/>
                Message: &nbsp;<span id="t_message"></span>
                <hr/>
                Error: &nbsp; <span id="t_error"></span>
                <hr/>
            </div>

            <div class="layui-tab-item">

                <table class="layui-table">
                    <colgroup>
                        <col width="50%">
                        <col width="30%">
                    </colgroup>
                    <thead>
                    <tr>
                        <th>地址</th>
                        <th>余额</th>
                        <th>操作</th>
                    </tr>
                    </thead>
                    <tbody id="balance_chart">
                        <tr>
                            <td>
                                <input type="text" name="account"   autocomplete="off" class="layui-input"
                                       placeholder="请输入查询地址" id="query_input_0">
                            </td>
                            <td id="query_balance_0">

                            </td>
                            <td>
                                <button class="layui-btn query_btn"  id="query_btn_0">查询</button>
                            </td>
                        </tr>
                        <tr>
                            <td>
                                <input type="text" name="account"   autocomplete="off" class="layui-input"
                                       placeholder="请输入查询地址" id="query_input_1">
                            </td>
                            <td id="query_balance_1">

                            </td>
                            <td>
                                <button class="layui-btn query_btn"  id="query_btn_1">查询</button>
                            </td>
                        </tr>
                        <tr>
                            <td>
                                <input type="text" name="account"   autocomplete="off" class="layui-input"
                                       placeholder="请输入查询地址" id="query_input_2">
                            </td>
                            <td id="query_balance_2">

                            </td>
                            <td>
                                <button class="layui-btn query_btn" id="query_btn_2">查询</button>
                            </td>
                        </tr>
                        <tr>
                            <td>
                                <input type="text" name="account"   autocomplete="off" class="layui-input"
                                       placeholder="请输入查询地址" id="query_input_3">
                            </td>
                            <td id="query_balance_3">

                            </td>
                            <td>
                                <button class="layui-btn query_btn" id="query_btn_3">查询</button>
                            </td>
                        </tr>
                        <tr>
                            <td>
                                <input type="text" name="account"   autocomplete="off" class="layui-input"
                                       placeholder="请输入查询地址" id="query_input_4">
                            </td>
                            <td id="query_balance_4">

                            </td>
                            <td>
                                <button class="layui-btn query_btn" id="query_btn_4">查询</button>
                            </td>
                        </tr>
                    </tbody>
                </table>
                <div style="padding-top: 30px"></div>
                <hr/>
                Message: &nbsp;<span id="balance_message"></span>
                <hr/>
                Error: &nbsp; <span id="balance_error"></span>
                <hr/>
            </div>
            <!-- 投票部分 -->
            <div class="layui-tab-item">
                <form class="layui-form" action="">
                    <div class="layui-form-item">
                        <label class="layui-form-label">from</label>
                        <div class="layui-input-block">
                            <input type="text" name="from"   autocomplete="off" class="layui-input"
                                   placeholder="请输入发送方地址(len=40)">
                        </div>
                    </div>
                    <div class="layui-form-item">
                        <label class="layui-form-label">TemplateName</label>
                        <div class="layui-input-block">
                            <input type="text" name="template_name"   autocomplete="off" class="layui-input"
                                   placeholder="" value="vote_template_1">
                        </div>
                    </div>
                    <div class="layui-form-item">
                        <label class="layui-form-label">PIndex</label>
                        <div class="layui-input-block">
                            <input type="text" name="p_index"   autocomplete="off" class="layui-input"
                                   placeholder="" value="2">
                        </div>
                    </div>
                    <div class="layui-form-item">
                        <label class="layui-form-label">PValue</label>
                        <div class="layui-input-block">
                            <input type="text" name="p_value"   autocomplete="off" class="layui-input"
                                   placeholder="" value="999">
                        </div>
                    </div>
                    <div class="layui-form-item">
                        <label class="layui-form-label">Custom</label>
                        <div class="layui-input-block">
                            <input type="radio" name="custom" value="true" title="true">
                            <input type="radio" name="custom" value="false" title="false" checked>
                        </div>
                    </div>
                    <div class="layui-form-item">
                        <label class="layui-form-label">Desc</label>
                        <div class="layui-input-block">
                            <input type="text" name="desc"   autocomplete="off" class="layui-input"
                                   placeholder="" value="描述">
                        </div>
                    </div>
                    <div class="layui-form-item">
                        <label class="layui-form-label">DepositMin</label>
                        <div class="layui-input-block">
                            <input type="text" name="deposit_min"   autocomplete="off" class="layui-input"
                                   placeholder="" value="1">
                        </div>
                    </div>
                    <div class="layui-form-item">
                        <label class="layui-form-label">TotalDepositMin</label>
                        <div class="layui-input-block">
                            <input type="text" name="total_deposit_min"   autocomplete="off" class="layui-input"
                                   placeholder="" value="2">
                        </div>
                    </div>
                    <div class="layui-form-item">
                        <label class="layui-form-label">VoterCntMin</label>
                        <div class="layui-input-block">
                            <input type="text" name="voter_cnt_min"   autocomplete="off" class="layui-input"
                                   placeholder="" value="4">
                        </div>
                    </div>
                    <div class="layui-form-item">
                        <label class="layui-form-label">ApprovalDepositMin</label>
                        <div class="layui-input-block">
                            <input type="text" name="approval_deposit_min"   autocomplete="off" class="layui-input"
                                   placeholder="" value="2">
                        </div>
                    </div>
                    <div class="layui-form-item">
                        <label class="layui-form-label">ApprovalVoterCntMin</label>
                        <div class="layui-input-block">
                            <input type="text" name="approval_voter_cnt_min"   autocomplete="off" class="layui-input"
                                   placeholder="" value="2">
                        </div>
                    </div>
                    <div class="layui-form-item">
                        <label class="layui-form-label">DeadlineBlock</label>
                        <div class="layui-input-block">
                            <input type="text" name="deadline_block"   autocomplete="off" class="layui-input"
                                   placeholder="" value="80">
                        </div>
                    </div>
                    <div class="layui-form-item">
                        <label class="layui-form-label">StatBlock</label>
                        <div class="layui-input-block">
                            <input type="text" name="stat_block"   autocomplete="off" class="layui-input"
                                   placeholder="" value="85">
                        </div>
                    </div>
                    <div class="layui-form-item">
                        <label class="layui-form-label">EffectBlock</label>
                        <div class="layui-input-block">
                            <input type="text" name="effect_block"   autocomplete="off" class="layui-input"
                                   placeholder="" value="90">
                        </div>
                    </div>
                    <div class="layui-form-item">
                        <label class="layui-form-label">DepositGap</label>
                        <div class="layui-input-block">
                            <input type="text" name="deposit_gap"   autocomplete="off" class="layui-input"
                                   placeholder="" value="1">
                        </div>
                    </div>
                    <div class="layui-form-item">
                        <div class="layui-input-block">
                            <button class="layui-btn" lay-submit lay-filter="vote_form">立即提交</button>
                            <button type="reset" class="layui-btn layui-btn-primary">重置</button>
                        </div>
                    </div>
                    <div style="padding-top: 30px"></div>
                    <hr/>
                    Message: &nbsp;<span id="vote_message"></span>
                    <hr/>
                    Error: &nbsp; <span id="vote_error"></span>
                    <hr/>
                </form>
            </div>
            <div class="layui-tab-item">
                <table id="block_detail" lay-filter="block_detail"></table>
            </div>
            <div class="layui-tab-item">
                <table id="group_detail" lay-filter="block_detail"></table>
            </div>
            <div class="layui-tab-item">
                <table class="layui-table">
                    <tr>
                        <td width="40%">
                            <input type="text" name="account"   autocomplete="off" class="layui-input"
                                   placeholder="根据高度查询工作组信息" id="query_wg_input">
                        </td>
                        <td>
                            <button class="layui-btn query_btn" id="query_wg_btn">查询</button>
                        </td>
                    </tr>
                </table>

                <table id="work_group_detail" lay-filter="block_detail"></table>
            </div>

            <!-- 共识验证部分 -->
            <div class="layui-tab-item">
                <table class="layui-table">
                    <tr>
                        <td width="40%">
                            <input type="text" name="consensus_stat"   autocomplete="off" class="layui-input"
                                   placeholder="根据高度统计共识算法执行情况" id="consensus_stat_input">
                        </td>
                        <td>
                            <button class="layui-btn query_btn" id="consensus_stat_btn">统计</button>
                        </td>
                    </tr>
                </table>
            </div>
        </div>
    </div>
</div>
<script src="http://www.ysh0566.top/js/layui/layui.js"></script>
<script>

    layui.use(['form', 'jquery', 'element', 'layer', 'table'], function(){
        var element = layui.element;
        var form = layui.form;
        var layer = layui.layer;
        var $ = layui.$;
        var HOST = "http://127.0.0.1:8088";
        var ref;
        var host_ele = $("#host");
        var online=false;
        var current_block_height = 0;
        host_ele.text(HOST);
        var blocks = [];
        var groups = [];
        var workGroups = [];
        var groupIds = new Set();
        var table = layui.table;


        var block_table = table.render({
            elem: '#block_detail' //指定原始表格元素选择器（推荐id选择器）
            ,cols: [[{field:'height',title: '块高', sort:true}, {field:'hash', title: 'hash'},{field:'pre_hash', title: 'pre_hash'},
                {field:'pre_time', title: 'pre_time', width: 189},{field:'queue_number', title: 'queue_number'},
                {field:'cur_time', title: 'cur_time', width: 189},{field:'castor', title: 'castor'},{field:'group_id', title: 'group_id'}, {field:'signature', title: 'signature'}]] //设置表头
            ,data: blocks
            ,page: true
            ,limit:15
        });

        var group_table = table.render({
            elem: '#group_detail' //指定原始表格元素选择器（推荐id选择器）
            ,cols: [[{field:'height',title: '高度', sort: true, width:140}, {field:'group_id',title: '组id', width:140}, {field:'dummy', title: 'dummy', width:80},
                {field:'parent', title: '父亲组', width:140},{field:'pre', title: '上一组', width:140},
                {field:'begin_height', title: '生效高度', width: 100},{field:'dismiss_height', title: '解散高度', width:100},
                {field:'members', title: '成员列表'}]] //设置表头
            ,data: groups
            ,page: true
            ,limit:15
        });

        var work_group_table = table.render({
            elem: '#work_group_detail' //指定原始表格元素选择器（推荐id选择器）
            ,cols: [[{field:'id',title: '组id', width:140}, {field:'parent', title: '父亲组', width:140},{field:'pre', title: '上一组', width:140},
                {field:'begin_height', title: '生效高度', width: 100},{field:'dismiss_height', title: '解散高度', width:100},
                {field:'group_members', title: '成员列表'}]] //设置表头
            ,data: groups
            ,page: true
            ,limit:15
        });


        $("#change_host").click(function () {
            layer.prompt({
                formType: 0,
                value: HOST,
                title: '请输入新的host',
            }, function(value, index, elem){
                HOST = value;
                host_ele.text(HOST);
                layer.close(index);
                current_block_height = 0;
                blocks = [];
                block_table.reload({
                    data: blocks
                });
            });
        });

        // 查询余额
        $(".query_btn").click(function () {
            let id = $(this).attr("id");
            let count = id.split("_")[2];
            $("#balance_message").text("");
            $("#balance_error").text("");
            let params = {
                "method": "GTAS_balance",
                "params": [$("#query_input_"+count).val()],
                "jsonrpc": "2.0",
                "id": "1"
            };
            $.ajax({
                type: 'POST',
                url: HOST,
                beforeSend: function (xhr) {
                    xhr.setRequestHeader("Content-Type", "application/json");
                },
                data: JSON.stringify(params),
                success: function (rdata) {
                    if (rdata.result !== undefined){
                        $("#balance_message").text(rdata.result.message);
                        $("#query_balance_"+count).text(rdata.result.data)
                    }
                    if (rdata.error !== undefined){
                        $("#balance_error").text(rdata.error.message);
                    }
                },
            });
        });

        // 钱包初始化
        function init_wallets() {
            let params = {
                "method": "GTAS_getWallets",
                "params": [],
                "jsonrpc": "2.0",
                "id": "1"
            };
            $.ajax({
                type: 'POST',
                url: HOST,
                beforeSend: function (xhr) {
                    xhr.setRequestHeader("Content-Type", "application/json");
                },
                data: JSON.stringify(params),
                success: function (rdata) {
                    if (rdata.result !== undefined){
                        $.each(rdata.result.data, function (i,val) {
                            let tr = "<tr class='wallet_tr'><td>" + val.private_key + "</td><td>" + val.address
                                    + '</td><td ><button class="layui-btn wallet_del">删除</button></td></tr>';
                            $("#create_chart").append(tr);

                            $(".wallet_del").click(function () {
                                let parent = $(this).parents("tr");
                                del_wallet(parent.children("td:eq(1)").text());
                                parent.remove();
                            });
                        })
                    }
                },
            });
        }

        function del_wallet(key) {
            let params = {
                "method": "GTAS_deleteWallet",
                "params": [key],
                "jsonrpc": "2.0",
                "id": "1"
            };
            $.ajax({
                type: 'POST',
                url: HOST,
                beforeSend: function (xhr) {
                    xhr.setRequestHeader("Content-Type", "application/json");
                },
                data: JSON.stringify(params),
                success: function (rdata) {

                },
            });
        }

        init_wallets();


        // 创建钱包
        $("#create_btn").click(function () {
            let params = {
                "method": "GTAS_newWallet",
                "params": [],
                "jsonrpc": "2.0",
                "id": "1"
            };
            $.ajax({
                type: 'POST',
                url: HOST,
                beforeSend: function (xhr) {
                    xhr.setRequestHeader("Content-Type", "application/json");
                },
                data: JSON.stringify(params),
                success: function (rdata) {
                    let tr = "<tr class='wallet_tr'><td>" + rdata.result.data.private_key + "</td><td>" + rdata.result.data.address
                            + '</td><td ><button class="layui-btn wallet_del">删除</button></td></tr>';
                    $("#create_chart").append(tr);

                    $(".wallet_del").click(function () {
                        let parent = $(this).parents("tr");
                        del_wallet(parent.children("td:eq(1)").text());
                        parent.remove();
                    });
                },
            });
        });

        // 投票表单提交
        form.on('submit(vote_form)', function (data) {
            $("#vote_message").text("");
            $("#vote_error").text("");
            let from = data.field.from;
            let vote_param = {};
            vote_param.template_name = data.field.template_name;
            vote_param.p_index = parseInt(data.field.p_index);
            vote_param.p_value = data.field.p_value;
            vote_param.custom = (data.field.custom === "true");
            vote_param.desc = data.field.desc;
            vote_param.deposit_min = parseInt(data.field.deposit_min);
            vote_param.total_deposit_min = parseInt(data.field.total_deposit_min);
            vote_param.voter_cnt_min = parseInt(data.field.voter_cnt_min);
            vote_param.approval_deposit_min = parseInt(data.field.approval_deposit_min);
            vote_param.approval_voter_cnt_min = parseInt(data.field.approval_voter_cnt_min);
            vote_param.deadline_block = parseInt(data.field.deadline_block);
            vote_param.stat_block = parseInt(data.field.stat_block);
            vote_param.effect_block = parseInt(data.field.effect_block);
            vote_param.deposit_gap = parseInt(data.field.deposit_gap);
            let params = {
                "method": "GTAS_vote",
                "params": [from, vote_param],
                "jsonrpc": "2.0",
                "id": "1"
            };
            $.ajax({
                type: 'POST',
                url: HOST,
                beforeSend: function (xhr) {
                    xhr.setRequestHeader("Content-Type", "application/json");
                },
                data: JSON.stringify(params),
                success: function (rdata) {
                    if (rdata.result !== undefined){
                        $("#vote_message").text(rdata.result.message)
                    }
                    if (rdata.error !== undefined){
                        $("#vote_error").text(rdata.error.message)
                    }
                },
            });
            return false;
        });


        // 交易表单提交
        form.on('submit(t_form)', function(data){
            $("#t_message").text("");
            $("#t_error").text("");
            let from = data.field.from;
            let to = data.field.to;
            let amount = data.field.amount;
            let code = data.field.code;
            // if (from.length !== 42) {
            //     layer.msg("from 参数字段长度错误");
            //     return false
            // }
            // if (to.length !== 42) {
            //     layer.msg("to 参数字段长度错误");
            //     return false
            // }
            let params = {
                "method": "GTAS_tx",
                "params": [from, to, parseFloat(amount), code],
                "jsonrpc": "2.0",
                "id": "1"
            };

            $.ajax({
                type: 'POST',
                url: HOST,
                beforeSend: function (xhr) {
                    xhr.setRequestHeader("Content-Type", "application/json");
                },
                data: JSON.stringify(params),
                success: function (rdata) {
                    if (rdata.result !== undefined){
                        $("#t_message").text(rdata.result.message)
                    }
                    if (rdata.error !== undefined){
                        $("#t_error").text(rdata.error.message)
                    }
                },
            });
            return false;
        });

        // 同步已链接节点
        function syncNodes() {
            let params = {
                "method": "GTAS_connectedNodes",
                "params": [],
                "jsonrpc": "2.0",
                "id": "1"
            };
            $.ajax({
                type: 'POST',
                url: HOST,
                beforeSend: function (xhr) {
                    xhr.setRequestHeader("Content-Type", "application/json");
                },
                data: JSON.stringify(params),
                success: function (rdata) {
                    if (rdata.result !== undefined){
                        let nodes_table = $("#nodes_table");
                        nodes_table.empty();
                        rdata.result.data.sort(function (a, b) {
                            return parseInt(a.ip.split(".")[3]) - parseInt(b.ip.split(".")[3])
                        });
                        $.each(rdata.result.data, function (i,val) {
                            nodes_table.append(
                                    " <tr><td>id</td><td>ip</td><td>port</td></tr>".replace("ip", val.ip).replace("id", val.id).replace("port", val.tcp_port)
                            )
                        })
                    }
                    if (rdata.error !== undefined){
                        // $("#t_error").text(rdata.error.message)
                    }
                },
                error: function () {
                    let nodes_table = $("#nodes_table");
                    nodes_table.empty();
                }
            });
        }

        // 同步缓冲区交易
        function syncTrans() {
            let params = {
                "method": "GTAS_transPool",
                "params": [],
                "jsonrpc": "2.0",
                "id": "1"
            };
            $.ajax({
                type: 'POST',
                url: HOST,
                beforeSend: function (xhr) {
                    xhr.setRequestHeader("Content-Type", "application/json");
                },
                data: JSON.stringify(params),
                success: function (rdata) {
                    if (rdata.result !== undefined){
                        let trans_table = $("#trans_table");
                        trans_table.empty();
                        rdata.result.data.sort(function (a, b) {
                            return parseInt(b.value) - parseInt(a.value)
                        });
                        $.each(rdata.result.data, function (i,val) {
                            trans_table.append(
                                    " <tr><td>source</td><td>target</td><td>value</td></tr>"
                                            .replace("value", val.value)
                                            .replace("source",  val.source.slice(0, 7) + "..." + val.source.slice(-6))
                                            .replace("target",  val.target.slice(0, 7) + "..." + val.target.slice(-6))
                            )
                        })
                    }
                    if (rdata.error !== undefined){
                        // $("#t_error").text(rdata.error.message)
                    }
                },
                error: function () {
                    let trans_table = $("#trans_table");
                    trans_table.empty();
                }
            });
        }

        // 同步块高
        function syncBlockHeight() {
            let params = {
                "method": "GTAS_blockHeight",
                "params": [],
                "jsonrpc": "2.0",
                "id": "1"
            };
            $.ajax({
                type: 'POST',
                url: HOST,
                beforeSend: function (xhr) {
                    xhr.setRequestHeader("Content-Type", "application/json");
                },
                data: JSON.stringify(params),
                success: function (rdata) {
                    if (rdata.result !== undefined){
                        let block_height = $("#block_height");
                        block_height.text(rdata.result.data)
                    }
                    if (rdata.error !== undefined){
                        // $("#t_error").text(rdata.error.message)
                    }
                    if(!online) {
                        online = true;
                        $("#online_flag").addClass("layui-bg-green")
                    }
                    if (current_block_height > rdata.result.data) {
                        current_block_height = 0;
                        blocks = [];
                        block_table.reload({
                            data: blocks
                        });
                    }
                    syncWorkGroupNum(rdata.result.data)
                    let count = 0;
                    for(let i=current_block_height+1; i<=rdata.result.data; i++) {
                        syncBlock(i);
                        count ++;
                        if (count >= 50) {
                            current_block_height = i;
                            return
                        }
                    }
                    current_block_height = rdata.result.data
                },
                error: function () {
                    if(online) {
                        online = false;
                        $("#online_flag").removeClass("layui-bg-green")
                    }
                }
            });
        }

        // 同步组高
        function syncGroupHeight() {
            let params = {
                "method": "GTAS_groupHeight",
                "params": [],
                "jsonrpc": "2.0",
                "id": "1"
            };
            $.ajax({
                type: 'POST',
                url: HOST,
                beforeSend: function (xhr) {
                    xhr.setRequestHeader("Content-Type", "application/json");
                },
                data: JSON.stringify(params),
                success: function (rdata) {
                    if (rdata.result !== undefined){
                        let block_height = $("#group_height");
                        block_height.text(rdata.result.data)
                        if (groups.length > 0 && rdata.result.data < groups[groups.length - 1]["height"]) {
                            groups = []
                            groupIds.clear()
                            syncGroup(0)
                        }
                    }
                    if (rdata.error !== undefined){
                        // $("#t_error").text(rdata.error.message)
                    }
                },
            });
        }

        // 同步组高
        function syncWorkGroupNum(height) {
            let params = {
                "method": "GTAS_workGroupNum",
                "params": [height],
                "jsonrpc": "2.0",
                "id": "1"
            };
            $.ajax({
                type: 'POST',
                url: HOST,
                beforeSend: function (xhr) {
                    xhr.setRequestHeader("Content-Type", "application/json");
                },
                data: JSON.stringify(params),
                success: function (rdata) {
                    if (rdata.result !== undefined){
                        let block_height = $("#work_group_num");
                        block_height.text(rdata.result.data)
                    }
                    if (rdata.error !== undefined){
                        // $("#t_error").text(rdata.error.message)
                    }
                },
            });
        }

        // 同步块信息
        function syncBlock(height) {
            let params = {
                "method": "GTAS_getBlock",
                "params": [height],
                "jsonrpc": "2.0",
                "id": "1"
            };
            $.ajax({
                type: 'POST',
                url: HOST,
                beforeSend: function (xhr) {
                    xhr.setRequestHeader("Content-Type", "application/json");
                },
                data: JSON.stringify(params),
                success: function (rdata) {
                    if (rdata.result !== undefined){
                        blocks.push(rdata.result.data);
                        block_table.reload({
                                    data: blocks
                                }
                        )
                    }
                    if (rdata.error !== undefined){
                        // $("#t_error").text(rdata.error.message)
                    }
                },
            });
        }

        // 同步组信息
        function syncGroup(height) {
            let params = {
                "method": "GTAS_getGroupsAfter",
                "params": [height],
                "jsonrpc": "2.0",
                "id": "1"
            };
            $.ajax({
                type: 'POST',
                url: HOST,
                beforeSend: function (xhr) {
                    xhr.setRequestHeader("Content-Type", "application/json");
                },
                data: JSON.stringify(params),
                success: function (rdata) {
                    if (rdata.result !== undefined && rdata.result != null && rdata.result.message == 'success'){
                        retArr = rdata.result.data
                        for(i = 0; i < retArr.length; i++) {
                            if (!groupIds.has(retArr[i]["group_id"])) {
                                groups.push(retArr[i])
                                groupIds.add(retArr[i]["group_id"])
                            }
                        }
                        group_table.reload({
                                    data: groups
                                }
                        )
                    }
                    if (rdata.error !== undefined){
                        // $("#t_error").text(rdata.error.message)
                    }
                },
                error: function (err) {
                    console.log(err)
                }
            });
        }

        $("#query_wg_btn").click(function () {
            var h = $("#query_wg_input").val()
            if (h == null || h == undefined || h == '') {
                alert("请输入查询高度")
                return
            }
            queryWorkGroup(parseInt(h))
        });
        //查询工作组
        function queryWorkGroup(height) {
            let params = {
                "method": "GTAS_getWorkGroup",
                "params": [height],
                "jsonrpc": "2.0",
                "id": "1"
            };
            $.ajax({
                type: 'POST',
                url: HOST,
                beforeSend: function (xhr) {
                    xhr.setRequestHeader("Content-Type", "application/json");
                },
                data: JSON.stringify(params),
                success: function (rdata) {
                    if (rdata.result !== undefined && rdata.result != null && rdata.result.message == 'success'){
                        retArr = rdata.result.data
                        work_group_table.reload({
                                    data: retArr
                                }
                        )
                    }
                    if (rdata.error !== undefined){
                        // $("#t_error").text(rdata.error.message)
                    }
                },
                error: function (err) {
                    console.log(err)
                }
            });
        }

        $("#consensus_stat_btn").click(function () {
            var h = $("#consensus_stat_input").val()
            if (h == null || h == undefined || h == '') {
                alert("请输入查询高度")
                return
            }
            doConsensusStat(parseInt(h))
        });

        function doConsensusStat(height){
            let params = {
                "method": "GTAS_consensusStat",
                "params": [height],
                "jsonrpc": "2.0",
                "id": "1"
            };
            $.ajax({
                type: 'POST',
                url: HOST,
                beforeSend: function (xhr) {
                    xhr.setRequestHeader("Content-Type", "application/json");
                },
                data: JSON.stringify(params),
                success: function (rdata) {
                    if (rdata.result !== undefined && rdata.result != null && rdata.result.message == 'success'){
                        alert("successs")
                    }
                    if (rdata.error !== undefined){
                        // $("#t_error").text(rdata.error.message)
                    }
                },
                error: function (err) {
                    console.log(err)
                }
            });
        }

        // dashboard同步数据
        syncNodes();
        syncTrans();
        syncBlockHeight();
        syncGroupHeight();
        syncGroup(0)
        setInterval(function () {
            if (groups.length > 0) {
                syncGroup(groups[groups.length-1]["height"]+1)
            } else {
                syncGroup(0)
            }
            syncBlockHeight();
        }, 1000);
        ref = setInterval(function(){
            syncNodes();
            syncTrans();
            syncGroupHeight();
        },1000);

        element.on('tab(demo)', function(data){
            if(data.index === 0) {
                ref = setInterval(function(){
                    syncNodes();
                    syncTrans();
                    syncGroupHeight();
                },1000);
            } else {
                clearInterval(ref)
            }
        });

    });
</script>

</body>
</html>`
