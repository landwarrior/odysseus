<?php

namespace App;

use Illuminate\Database\Eloquent\Model;

class MstHr extends Model
{
    /** テーブル名 */
    protected $table = 'mst_hr';
    /** プライマリキー */
    protected $primaryKey = 'hr_cd';
}
