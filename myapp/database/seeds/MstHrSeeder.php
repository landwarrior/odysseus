<?php

use Illuminate\Database\Seeder;
use App\MstHr;
use Illuminate\Support\Facades\Hash;

class MstHrSeeder extends Seeder
{
    /**
     * Run the database seeds.
     *
     * @return void
     */
    public function run()
    {
        $hr = [
            'hr_cd' => 'root',
            'user_name' => 'root',
            'name_kana' => 'root',
            'password' => Hash::make('rootroot'),
            'is_admin' => 1,
            'remarks' => 'default root user',
        ];
        MstHr::create($hr);
    }
}
