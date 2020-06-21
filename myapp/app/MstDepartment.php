<?php

namespace App;

use Illuminate\Database\Eloquent\Model;

class MstDepartment extends Model
{
    /** テーブル名 */
    protected $table = 'mst_department';
    /** プライマリキー */
    protected $primaryKey = 'department_id';
    /** IDが自動増分されるか */
    public $incrementing = false;
}
