<?php

namespace App;

use Illuminate\Database\Eloquent\Model;

class MstBp extends Model
{
    /** テーブル名 */
    protected $table = 'mst_bp';
    /** プライマリキー */
    protected $primaryKey = 'bp_id';
    /** IDが自動増分されるか */
    public $incrementing = false;
}
