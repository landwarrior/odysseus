<?php

namespace App;

use Illuminate\Database\Eloquent\Model;

class MstRole extends Model
{
    /** テーブル名 */
    protected $table = 'mst_role';
    /** プライマリキー */
    protected $primaryKey = 'role_id';
    /** IDが自動増分されるか */
    public $incrementing = false;
}
