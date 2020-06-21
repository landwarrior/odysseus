<?php

namespace App;

use Illuminate\Database\Eloquent\Model;

class TrnProjectDetail extends Model
{
    /** テーブル名 */
    protected $table = 'trn_project_detail';
    /** プライマリキー */
    protected $primaryKey = ['project_no', 'process_id'];
    /** IDが自動増分されるか */
    public $incrementing = false;
    /** 日付データ */
    protected $dates = ['from_date', 'to_date'];
}
