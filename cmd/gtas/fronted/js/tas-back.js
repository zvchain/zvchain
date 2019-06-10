
layui.use(['form', 'jquery', 'element', 'layer', 'table'], function(){
    var element = layui.element;
    var form = layui.form;
    var layer = layui.layer;
    var $ = layui.$;
    var HOST = "/";
    var dashboard;
    var host_ele = $("#host");
    var online=false;
    var current_block_height = 0;
    host_ele.text(HOST);
    // var blocks = [];
    // var groups = [];
    var workGroups = [];
    var groupIds = new Set();
    var table = layui.table;


    var block_table = table.render({
        elem: '#block_detail', //指定原始表格元素选择器（推荐id选择器）
        initSort: {
            field: 'height',
            type: 'asc'
        }
        ,url: '/page'
        ,where:{
            "method": "GTAS_pageGetBlocks",
            "params": [],
            "jsonrpc": "2.0",
            "id": "1"
        }
        ,method: "post"
        ,contentType: 'application/json'
        ,parseData: function (res) {
            return {
                "code": 0,
                "msg": res.result.message,
                "count": res.result.data.count,
                "data": res.result.data.data
            };
        }
        ,cols: [[{field:'height',title: '块高', sort:true}, {field:'hash', title: 'hash'},{field:'pre_hash', title: 'pre_hash'},
            {field:'pre_time', title: 'pre_time', width: 189},
            {field:'cur_time', title: 'cur_time', width: 189},{field:'castor', title: 'castor'},{field:'group_id', title: 'group_id'},{toolbar:'#block_detail_view'}]] //设置表头
        ,page: true
        ,limit:15
    });

    table.on('tool(block_detail)', function (obj) {
        var data = obj.data; //获得当前行数据
        var layEvent = obj.event; //获得 lay-event 对应的值（也可以是表头的 event 参数对应的值）
        if (layEvent == 'detail') {
            let params = {
                "method": "GTAS_blockDetail",
                "params": [data.hash],
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
                success: function(str_response) {
                    var obj = window.open("about:blank");
                    obj.document.write(JSON.stringify(str_response, null, 4));
                }
            });
        }
    });

    var group_table = table.render({
        elem: '#group_detail' //指定原始表格元素选择器（推荐id选择器）
        ,initSort: {
            field: 'begin_height',
            type: 'asc'
        }
        ,url: '/page'
        ,where:{
            "method": "GTAS_pageGetGroups",
            "params": [],
            "jsonrpc": "2.0",
            "id": "1"
        }
        ,method: "post"
        ,contentType: 'application/json'
        ,cols: [[{field:'height',title: '高度', sort: true, width:140}, {field:'id',title: '组id', width:140},
            {field:'pre_id', title: '上一组', width:140},{field:'parent_id', title: '父亲组', width:140},
            {field:'begin_height', title: '生效高度', sort:true,width: 100},{field:'dismiss_height', title: '解散高度', width:100},
            {field:'members', title: '成员列表'}]] //设置表头
        ,parseData: function (res) {
            return {
                "code": 0,
                "msg": res.result.message,
                "count": res.result.data.count,
                "data": res.result.data.data
            };
        }
        ,page: true
        ,limit:15
    });

    var work_group_table = table.render({
        elem: '#work_group_detail' //指定原始表格元素选择器（推荐id选择器）
        ,cols: [[{field:'id',title: '组id', width:140}, {field:'parent', title: '父亲组', width:140},{field:'pre', title: '上一组', width:140},
            {field:'begin_height', title: '生效高度', width: 100},{field:'dismiss_height', title: '解散高度', width:100},
            {field:'group_members', title: '成员列表'}]] //设置表头
        ,data: workGroups
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
        let private_key = data.field.private_key;
        let to = data.field.to;
        let value = data.field.value;
        let txdata = data.field.data;
        let t = data.field.type;
        let nonce = data.field.nonce;
        let gas = data.field.gas;
        let gas_price = data.field.gas_price;
        // if (from.length !== 42) {
        //     layer.msg("from 参数字段长度错误");
        //     return false
        // }
        // if (to.length !== 42) {
        //     layer.msg("to 参数字段长度错误");
        //     return false
        // }
        //func (api *GtasAPI) TxUnSafe(privateKey, target string, value, gas, gasprice, nonce uint64, txType int, data string) (*Result, error) {

        let params = {
            "method": "GTAS_txUnSafe",
            "params": [private_key, to, parseFloat(value), parseInt(gas),parseInt(gas_price), parseInt(nonce), parseInt(t),txdata],
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
                    // blocks = [];
                    // block_table.reload({
                    //     data: blocks
                    // });
                }
                syncWorkGroupNum(rdata.result.data);
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
                    // if (groups.length > 0 && rdata.result.data < groups[groups.length - 1]["height"]) {
                    //     groups = []
                    //     groupIds.clear()
                    //     syncGroup(0)
                    // }
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
        // let params = {
        //     "method": "GTAS_getBlock",
        //     "params": [height],
        //     "jsonrpc": "2.0",
        //     "id": "1"
        // };
        // $.ajax({
        //     type: 'POST',
        //     url: HOST,
        //     beforeSend: function (xhr) {
        //         xhr.setRequestHeader("Content-Type", "application/json");
        //     },
        //     data: JSON.stringify(params),
        //     success: function (rdata) {
        //         if (rdata.result !== undefined){
        //             blocks.push(rdata.result.data);
        //             block_table.reload({
        //                     data: blocks
        //                 }
        //             )
        //         }
        //         if (rdata.error !== undefined){
        //             // $("#t_error").text(rdata.error.message)
        //         }
        //     },
        // });
    }

    // 同步组信息
    function syncGroup(height) {
        // let params = {
        //     "method": "GTAS_getGroupsAfter",
        //     "params": [height],
        //     "jsonrpc": "2.0",
        //     "id": "1"
        // };
        // $.ajax({
        //     type: 'POST',
        //     url: HOST,
        //     beforeSend: function (xhr) {
        //         xhr.setRequestHeader("Content-Type", "application/json");
        //     },
        //     data: JSON.stringify(params),
        //     success: function (rdata) {
        //         if (rdata.result !== undefined && rdata.result != null && rdata.result.message == 'success'){
        //             retArr = rdata.result.data
        //             for(i = 0; i < retArr.length; i++) {
        //                 if (!groupIds.has(retArr[i]["group_id"])) {
        //                     groups.push(retArr[i])
        //                     groupIds.add(retArr[i]["group_id"])
        //                 }
        //             }
        //             group_table.reload({
        //                     data: groups
        //                 }
        //             )
        //         }
        //         if (rdata.error !== undefined){
        //             // $("#t_error").text(rdata.error.message)
        //         }
        //     },
        //     error: function (err) {
        //         console.log(err)
        //     }
        // });
    }

    $("#query_wg_btn").click(function () {
        var h = $("#query_wg_input").val();
        if (h == null || h == undefined || h == '') {
            alert("请输入查询高度");
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
                    retArr = rdata.result.data;
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

    $(document).on("click", "a[name='miner_oper_a']", function () {
        m = $(this).attr("method");
        t = $(this).attr("mtype");
        let params = {
            "method": m,
            "params": [parseInt(t)],
            "jsonrpc": "2.0",
            "id": "1"
        };
        text = $(this).text();
        $.ajax({
            type: 'POST',
            url: HOST,
            beforeSend: function (xhr) {
                xhr.setRequestHeader("Content-Type", "application/json");
            },
            data: JSON.stringify(params),
            success: function (rdata) {
                if (rdata.result.message == "success") {
                    $(this).text("已申请"+text)
                } else {
                    alert(rdata.result.message)
                }
            },
            error: function (err) {
                console.log(err)
            }
        });
    });
    //查询工作组
    function queryNodeInfo() {
        let params = {
            "method": "GTAS_nodeInfo",
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
                if (rdata.result !== undefined && rdata.result != null && rdata.result.message == 'success'){
                    d = rdata.result.data;
                    $("#tb_node_id").text(d.id);
                    $("#tx_send_from").val(d.id);
                    $("#tb_node_balance").text(d.balance);
                    $("#tb_node_status").text(d.status);
                    $("#tb_node_type").text(d.n_type);
                    $("#tb_node_wg").text(d.w_group_num);
                    $("#tb_node_ag").text(d.a_group_num);
                    $("#tb_stake_body").html("");
                    $.each(d.mort_gages, function (i, v) {
                        tr = "<tr>";
                        tr += "<td>" + v.stake + "</td>";
                        tr += "<td>" + v.type + "</td>";
                        tr += "<td>" + v.apply_height + "</td>";
                        tr += "<td>" + v.abort_height + "</td>";
                        mtype = 0;
                        if (v.type == "重节点") {
                            mtype = 1
                        }
                        if (v.abort_height > 0) {
                            tr += "<td><a href=\"javascript:void(0);\" name='miner_oper_a' method='GTAS_minerRefund' mtype=" + mtype + ">退款</a></td>"
                        } else {
                            tr += "<td><a href=\"javascript:void(0);\" name='miner_oper_a' method='GTAS_minerAbort' mtype=" + mtype + ">取消</a></td>"
                        }
                        tr += "</tr>";
                        $("#tb_stake_body").append(tr)
                    })

                }
            },
            error: function (err) {
                console.log(err);
                $("#tb_node_status").text("已停止")
            }
        });
    }

    $("#apply_a").click(function () {
        f = $("#apply_miner_div");
        if(f.is(":hidden")){
            $(this).text("取消申请");
            f.show()
        }else{
            $(this).text("申请成为矿工");
            f.hide()
        }

    });

    $("#submit_apply").click(function () {
        stake = parseInt($("#text_stake").val());
        t = parseInt($("input[name='app_type_rd']:checked").val());
        $("#submit_result").text("");
        let params = {
            "method": "GTAS_minerApply",
            "params": [stake, t],
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
                $("#submit_result").text(rdata.result.message)
            },
            error: function (err) {
                console.log(err)
            }
        });
    });

    $("#consensus_stat_btn").click(function () {
        var h = $("#consensus_stat_input").val();
        if (h == null || h == undefined || h == '') {
            alert("请输入查询高度");
            return
        }
        doConsensusStat(parseInt(h))
    });

    function doConsensusStat(height) {
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
                if (rdata.result !== undefined && rdata.result != null && rdata.result.message == 'success') {
                    alert("successs")
                }
                if (rdata.error !== undefined) {
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
    // syncGroup(0)
    queryNodeInfo();
    setInterval(function () {
        queryNodeInfo()
    }, 3000);
    setInterval(function () {
        // if (groups.length > 0) {
        //     syncGroup(groups[groups.length-1]["height"]+1)
        // } else {
        //     syncGroup(0)
        // }
        syncBlockHeight();
    }, 1000);
    dashboard = setInterval(function(){
        syncNodes();
        syncTrans();
        syncGroupHeight();
    },1000);

    blocktable_inter = 0;
    grouptable_inter = 0;

    element.on('tab(demo)', function(data){
        if(data.index === 0) {
            dashboard = setInterval(function(){
                syncNodes();
                syncTrans();
                syncGroupHeight();
            },1000);
        } else {
            clearInterval(dashboard)
        }
        if (data.index == 5) {
            blocktable_inter = setInterval(function () {
                block_table.reload()
            }, 2000);
        } else {
            clearInterval(blocktable_inter)
        }
        if (data.index == 6) {
            grouptable_inter = setInterval(function () {
                group_table.reload()
            }, 2000);
        } else {
            clearInterval(grouptable_inter)
        }

    });

});