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
            $sum_price = 0;
            foreach ($persons as $k => $val) {
              $sum_price += $val->result_hour * $val->price;
            }
            $results[] = [
                'project_no' => $value->project_no,
                'name' => $value->name,
                'order_amount' => $value->order_amount,
                'sum_result' => $sum_price,
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
  , p.to_date
  , ifnull(sum(d.man_per_day), 0) as man_per_day_sum
  , (
    select distinct
      sum(r.result_hour) as result_hour_sum
    from
      trn_hr_result r
    where
      r.project_no = p.project_no
    group by
      r.project_no
  ) as result_hour_sum
from
  trn_project p
  left outer join trn_project_detail d
    on d.project_no = p.project_no
where
  p.is_finished = 0
group by
  p.project_no
  , p.name
  , p.order_amount
  , p.to_date
order by
  ifnull(p.to_date, '9999-12-31')
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
  , r.target_date
  , r.result_hour
  , (
    select
      min(price)
    from
      mst_hr_unit_price
    where
      hr_cd = r.hr_cd
      and role_id = h.role_id
      and from_date <= r.target_date
    group by
      hr_cd
      , role_id
  ) as price
from
  trn_hr_result r
  inner join trn_project_detail_hr h
    on h.project_no = r.project_no
    and h.process_id = r.process_id
    and h.hr_cd = r.hr_cd
where
 r.project_no = '{$project_no}'
SQL;
        $results = DB::select($sql);

        return $results;
    }
}
