<?php

namespace App;

use Illuminate\Database\Eloquent\Model;

class MstProcess extends Model
{
    /** テーブル名 */
    protected $table = 'mst_process';
    /** プライマリキー */
    protected $primaryKey = 'process_id';
    /** IDが自動増分されるか */
    public $incrementing = false;
}
