<?php

use Illuminate\Database\Seeder;
use App\MstProcess;

class MstProcessSeeder extends Seeder
{
    /**
     * Run the database seeds.
     *
     * @return void
     */
    public function run()
    {
        $processes = [
            ['process_id' => 10, 'name' => '提案'],
            ['process_id' => 20, 'name' => '要件定義'],
            ['process_id' => 30, 'name' => '基本設計'],
            ['process_id' => 40, 'name' => '詳細設計'],
            ['process_id' => 50, 'name' => '製造'],
            ['process_id' => 60, 'name' => '単体テスト'],
            ['process_id' => 70, 'name' => '結合テスト'],
            ['process_id' => 80, 'name' => 'シナリオテスト'],
        ];
        foreach($processes as $process) {
            MstProcess::create($process);
        }
    }
}
