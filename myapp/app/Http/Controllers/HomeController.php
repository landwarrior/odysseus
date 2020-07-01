<?php

namespace App\Http\Controllers;

use Illuminate\Http\Request;
use Illuminate\Support\Facades\Auth;
use Illuminate\Support\Facades\DB;

class HomeController extends Controller
{
    /**
     * Create a new controller instance.
     *
     * @return void
     */
    public function __construct()
    {
        $this->middleware('auth');
    }

    /**
     * Show the application dashboard.
     *
     * @return \Illuminate\Contracts\Support\Renderable
     */
    public function index()
    {
        $user = Auth::user();
        $data = ProjectService::getProjectSummary();
        $results = [];
        foreach ($data as $key => $value) {
            $persons = ProjectService::getPersonResult($value->project_no);
            // foreach ($persons as $k => $val) {
            //
            // }
            $results[] = [
                'project_no' => $value->project_no,
                'name' => $value->name,
                'order_amount' => $value->order_amount,
                'to_date' => str_replace('-', '/', $value->to_date),
                'man_day' => $value->man_per_day_sum,
                'result_day' => round($value->result_hour_sum/8*10)/10
            ];
        }
        return view(
            'home',
            ['is_admin' => $user->is_admin, 'results' => $results]
        );
    }
}

class ProjectService
{
    public static function getProjectSummary()
    {
        $sql = <<<SQL
select
  p.project_no
  , p.name
  , ifnull(p.order_amount, 0) as order_amount
  , p.from_date
  , p.to_date
  , ifnull(sum(d.man_per_day), 0) as man_per_day_sum
  , ifnull(sum(r.result_hour), 0) as result_hour_sum
from
  trn_project p
  inner join trn_project_detail d
    on d.project_no = d.project_no
  left outer join trn_hr_result r
    on r.project_no = p.project_no
where
  p.is_finished = 0
group by
  p.project_no
  , p.name
  , p.order_amount
  , p.from_date
  , p.to_date
order by
  p.from_date
SQL;
        $results = DB::select($sql);

        return $results;
    }

    public static function getPersonResult($project_no)
    {
        $sql = <<<SQL
select
  r.project_no
  , r.hr_cd
  , h.role_id
  , sum(r.result_hour)
from
  trn_hr_result r
  inner join trn_project_detail_hr h
    on h.project_no = r.project_no
    and h.process_id = r.process_id
    and h.hr_cd = r.hr_cd
where
  r.project_no = '{$project_no}'
group by
  r.project_no
  , r.hr_cd
  , h.role_id
SQL;
        $results = DB::select($sql);

        return $results;
    }
}
