<?php

namespace App;

use Illuminate\Database\Eloquent\Model;

class MstHrUnitPrice extends Model
{
    /** テーブル名 */
    protected $table = 'mst_hr_unit_price';
    /** プライマリキー */
    protected $primaryKey = ['hr_cd', 'from_date', 'role_id'];
    /** IDが自動増分されるか */
    public $incrementing = false;
    /** 日付データ */
    protected $dates = ['from_date'];
}
