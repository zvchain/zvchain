
layui.use(['form', 'jquery', 'element', 'layer', 'table'], function(){
    var $ = layui.$;
    var HOST = "/";
    var element = layui.element;
    var form = layui.form;
    var layer = layui.layer;
    var table = layui.table;

    var recent_query = [];

    function addRecentQuery(q) {
        if ($.inArray(q, recent_query) >= 0) {
            return
        }
        if (recent_query.length >= 10) {
            recent_query.shift()
        }
        recent_query.push(q);
        pan = $("#recent_query_block");
        pan.html('');
        for (i = recent_query.length-1; i >=0; i--) {
            pan.append('<div class="layui-col-md1"><a href="javascript:void(0);" name="a_recent_query" style="color: #01AAED;" hash="' + recent_query[i] + '">' + recent_query[i].substr(0, 10) + '</a></div>')
        }
    }
    $(document).on("click", "a[name='a_recent_query']", function () {
        h = $(this).attr("hash");
        queryBlockDetail(h)
    });
    $(document).on("click", "a[name='block_table_hash_row']", function () {
        element.tabChange("demo", "block_detail_tab");
        h = $(this).text();
        queryBlockDetail(h)
    });
    $(document).on("click", "a[name='bonus_table_hash_row']", function () {
        h = $(this).text();
        queryBlockDetail(h)
    });

    $(document).on("click", "a[name='tx_hash_row']", function () {
        element.tabChange("demo", "tx_detail_tab");
        h = $(this).text();
        queryTxDetail(h)
    });

    function queryBlockDetail(h) {
        $("#query_block_hash").val(h);
        let params = {
            "method": "Dev_blockDetail",
            "params": [h],
            "jsonrpc": "2.0",
            "id": "1"
        };
        $("#block_detail_result").hide();
        $.ajax({
            type: 'POST',
            url: HOST,
            beforeSend: function (xhr) {
                xhr.setRequestHeader("Content-Type", "application/json");
            },
            data: JSON.stringify(params),
            success: function (rdata) {
                if (rdata.result.message != "success") {
                    alert(rdata.result.message);
                    return
                }
                $("#block_detail_result").show();
                d = rdata.result.data;
                $("#block_detail_height").text(d.height);
                $("#block_castor").text(d.castor);
                $("#block_hash").text(d.hash);
                $("#block_pre_hash").text(d.pre_hash);
                $("#block_ts").text(d.cur_time);
                $("#block_pre_ts").text(d.pre_time);
                $("#block_group").text(d.group_id);
                $("#block_tx_cnt").text(d.txs.length);
                $("#block_qn").text(d.qn);
                $("#block_total_qn").text(d.total_qn);
                $("#block_pre_total_qn").text(d.pre_total_qn);

                gbt = d.gen_bonus_tx;
                if (gbt != null && gbt != undefined) {
                    $("#gen_bonus_hash").text(gbt.hash);
                    $("#gen_bouns_value").text(gbt.value);
                    target = $("#gen_bonus_targets");
                    target.html('');
                    $.each(gbt.target_ids, function (i, v) {
                        target.append('<div class="layui-row">' + v + '</div>')
                    })
                } else {
                    $("#gen_bonus_hash").text('--');
                    $("#gen_bouns_value").text('--');
                    $("#gen_bonus_targets").html('--')
                }
                table.render({
                    elem: '#bonus_balance_table' //指定原始表格元素选择器（推荐id选择器）
                    ,cols: [[{field:'id',title: '矿工id'}, {field:'explain', title: '奖励类型'},{field:'pre_balance', title: '前块余额'}
                        ,{field:'curr_balance', title: '当前余额'},{field:'expect_balance', title: '期望余额'},{field:'expect_balance', title: '结果', templet: function (d) {
                                if (d.expect_balance == d.curr_balance) {
                                    return '<span style="color: green;">正确</span>'
                                } else {
                                    return '<span style="color: red;">错误</span>'
                                }
                            }}]] //设置表头
                    ,data: d.miner_bonus
                });
                table.render({
                    elem: '#bonus_table' //指定原始表格元素选择器（推荐id选择器）
                    ,cols: [[{field:'hash',title: 'hash',templet: '<div><a href="javascript:void(0);" class="layui-table-link" name="tx_hash_row">{{d.hash}}</a></div>' },
                        {field:'block_hash',title: '块hash', templet: '<div><a href="javascript:void(0);" class="layui-table-link" name="bonus_table_hash_row">{{d.block_hash}}</a></div>'},
                        {field:'value', title: '奖励', width:80},{field:'status_report', title: '状态', width:80},{field:'group_id', title: '组id'},
                        {field:'target_ids', title: '目标id列表'}]] //设置表头
                    ,data: d.body_bonus_txs
                });

                table.render({
                    elem: '#txs_table' //指定原始表格元素选择器（推荐id选择器）
                    ,cols: [[{field:'hash',title: 'hash', sort:true, templet: '<div><a href="javascript:void(0);" class="layui-table-link" name="tx_hash_row">{{d.hash}}</a></div>'}, {field:'type', title: '类型'},{field:'source', title: '来源'}
                        ,{field:'target', title: '目标'},{field:'value', title: '金额'}]] //设置表头
                    ,data: d.trans,
                    page: true,
                    limit: 20
                });

                addRecentQuery(h)
            },
            error: function (err) {
                console.log(err)
            }
        });
    }

    function queryTxDetail(h) {
        $("#query_tx_hash").val(h);
        let params = {
            "method": "Dev_getTransaction",
            "params": [h],
            "jsonrpc": "2.0",
            "id": "1"
        };
        $("#tx_detail_set").hide();
        $("#tx_receipt_set").hide();
        $.ajax({
            type: 'POST',
            url: HOST,
            beforeSend: function (xhr) {
                xhr.setRequestHeader("Content-Type", "application/json");
            },
            data: JSON.stringify(params),
            success: function (rdata) {
                if (rdata.result.message != "success") {
                    alert(rdata.result.message);
                    return
                }

                $("#tx_detail_set").show();
                d = rdata.result.data;

                $("#tx_hash").text(d.hash);
                $("#tx_source").text(d.source);
                $("#tx_target").text(d.target);
                $("#tx_value").text(d.value)



            },
            error: function (err) {
                console.log(err)
            }
        });

        let params2 = {
            "method": "Gtas_txReceipt",
            "params": [h],
            "jsonrpc": "2.0",
            "id": "1"
        };
        $.ajax({
            type: 'POST',
            url: HOST,
            beforeSend: function (xhr) {
                xhr.setRequestHeader("Content-Type", "application/json");
            },
            data: JSON.stringify(params2),
            success: function (rdata) {
                if (rdata.result.message != "success") {
                    // alert(rdata.result.message)
                    return
                }
                $("#tx_receipt_set").show();
                d = rdata.result.data.Receipt;
                t = rdata.result.data.Transaction;

                $("#tx_data").text(t.Data);

                $("#tx_cumulativeGasUsed").text(d.cumulativeGasUsed);
                $("#tx_logs").text(d.logs);
                $("#tx_contractAddress").text(d.contractAddress);
                var t_type = '';
                switch (t.Type) {
                    case 1:
                        t_type = "转账";
                        break;
                    case 2:
                        t_type = "合约创建";
                        break;
                    case 3:
                        t_type = "分红交易";
                        break;
                    case 4:
                        t_type = "矿工申请";
                        break;
                    case 5:
                        t_type = "矿工取消";
                        break;
                    case 6:
                        t_type = "取回质押";
                        break;
                }
                $("#tx_type").text(t_type);
                var t_status = '';
                switch (d.status) {
                    case 1:
                        t_status = "成功";
                        break;
                    case 0:
                        t_status = "失败";
                        break;
                }
                $("#tx_status").text(t_status)

            },
            error: function (err) {
                console.log(err)
            }
        });
    }

    $("#query_block_btn").click(function () {
        h = $("#query_block_hash").val();
        if (h == '')
            return;
        queryBlockDetail(h)
    });

    $("#query_tx_btn").click(function () {
        h = $("#query_tx_hash").val();
        if (h == '')
            return;
        queryTxDetail(h)
    })

});