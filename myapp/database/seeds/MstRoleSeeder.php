<?php

use Illuminate\Database\Seeder;
use App\MstRole;

class MstRoleSeeder extends Seeder
{
    /**
     * Run the database seeds.
     *
     * @return void
     */
    public function run()
    {
        $roles = [
            ['role_id' => 1, 'name' => 'PG'],
            ['role_id' => 2, 'name' => 'SE'],
            ['role_id' => 3, 'name' => 'PL'],
            ['role_id' => 4, 'name' => 'PM'],
        ];
        foreach($roles as $role) {
            MstRole::create($role);
        }
    }
}
