<?php

namespace App;

use Illuminate\Database\Eloquent\Model;

class TrnProjectDetailHr extends Model
{
    /** テーブル名 */
    protected $table = 'trn_project_detail_hr';
    /** プライマリキー */
    protected $primaryKey = ['project_no', 'process_id', 'hr_cd'];
}
