<?php

namespace App;

use Illuminate\Foundation\Auth\User as Authenticatable;
use Illuminate\Notifications\Notifiable;

class MstHr extends Authenticatable
{
    // 認証に使うテーブルとする
    use Notifiable;
    /** テーブル名 */
    protected $table = 'mst_hr';
    /** プライマリキー */
    protected $primaryKey = 'hr_cd';
    /** IDが自動増分されるか */
    public $incrementing = false;
    /** 任意に更新可能とする属性を指定 */
    protected $fillable = [
        'hr_cd',
        'name',
        'name_kana',
        'is_admin',
        'remarks',
        'bp_id',
    ];
    /** たぶん、見せたくない項目を定義 */
    protected $hidden = [
        'password', 'remember_token',
    ];
}
