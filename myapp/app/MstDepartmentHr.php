<?php

namespace App;

use Illuminate\Database\Eloquent\Model;

class MstDepartmentHr extends Model
{
    /** テーブル名 */
    protected $table = 'mst_department_hr';
    /** プライマリキー */
    protected $primaryKey = ['department_id', 'hr_cd'];
    /** IDが自動増分されるか */
    public $incrementing = false;
}
