<?php

namespace App;

use Illuminate\Database\Eloquent\Model;

class TrnHrResult extends Model
{
    /** テーブル名 */
    protected $table = 'trn_hr_result';
    /** プライマリキー */
    protected $primaryKey = ['project_no', 'process_id', 'hr_cd', 'target_date'];
    /** IDが自動増分されるか */
    public $incrementing = false;
}
