<?php

namespace App;

use Illuminate\Database\Eloquent\Model;

class TrnProject extends Model
{
    /** テーブル名 */
    protected $table = 'trn_project';
    /** プライマリキー */
    protected $primaryKey = 'project_no';
    /** IDが自動増分されるか */
    public $incrementing = false;
    /** 日付データ */
    protected $dates = ['from_date', 'to_date'];
}
