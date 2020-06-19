<?php

namespace App;

use Illuminate\Database\Eloquent\Model;

class MstHrUnitPrice extends Model
{
    /** テーブル名 */
    protected $table = 'mst_hr_unit_price';
    /** プライマリキー */
    protected $primaryKey = ['hr_cd', 'from_date', 'role_id'];
    /** 日付データ */
    protected $dates = ['from_date'];
}
