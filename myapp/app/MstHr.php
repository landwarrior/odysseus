<?php

namespace App;

use Illuminate\Database\Eloquent\Model;
use Illuminate\Foundation\Auth\User as Authenticatable;
use Illuminate\Notifications\Notifiable;

class MstHr extends Model
{
    // 認証に使うテーブルとする
    use Notifiable;
    /** テーブル名 */
    protected $table = 'mst_hr';
    /** プライマリキー */
    protected $primaryKey = 'hr_cd';
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
