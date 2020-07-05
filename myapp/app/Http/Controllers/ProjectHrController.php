<?php

namespace App\Http\Controllers;

use Illuminate\Support\MessageBag;
use Illuminate\Http\Request;
use App\MstProcess;
use App\MstRole;
use App\TrnProject;
use App\TrnProjectDetail;
use App\TrnProjectDetailHr;
use Illuminate\Support\Facades\Auth;
use Illuminate\Support\Facades\DB;
use App\Http\Requests\ProjectHrRequest;

class ProjectHrController extends Controller
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

    public function edit($project_no)
    {
        $user = Auth::user();
        if (!$user->is_admin) {
            return redirect(route('home'))->with('not_admin', '1');
        }
        $project = TrnProject::findOrFail($project_no);
        $processes = MstProcess::all();
        $roles = MstRole::all();
        $projectDetail = TrnProjectDetail::where('project_no', $project_no)->get();
        $projectDetailHrs = TrnProjectDetailHr::where('project_no', $project_no)->get();
        $pd = [];
        foreach ($projectDetail as $detail) {
            $process_name = '';
            $selected_hr = [];
            // 工程名の取得
            foreach ($processes as $process) {
                if ($process->process_id == $detail->process_id) {
                    $process_name = $process->name;
                    break;
                }
            }
            // 箱を用意
            foreach ($roles as $role) {
                $selected_hr[$role->name] = [];
            }
            // ユーザーコードの取得
            foreach ($projectDetailHrs as $selected) {
                if ($selected->process_id == $detail->process_id) {
                    $role_name = "";
                    foreach ($roles as $role) {
                        if ($role->role_id == $selected->role_id) {
                            $role_name = $role->name;
                        }
                    }
                    $selected_hr[$role_name][] = $selected->hr_cd;
                }
            }
            array_push(
                $pd,
                [
                    'process_id' => $detail->process_id,
                    'process_name' => $process_name,
                    'man_per_day' => $detail->man_per_day,
                    'selected' => $selected_hr,
                ]
            );
        }
        $hrs = [];
        foreach ($roles as $role) {
            if (!isset($hrs[$role->name])) {
                $hrs[$role->name] = [];
            }
            $from = date('Y-m-d');
            if (!empty($project->from_date)) {
              $from = $project->from_date->format('Y-m-d');
            }
            $data = ProjectHrService::getEnableHrs($from, $role->role_id);
            foreach ($data as $key=>$value) {
                $hrs[$role->name][] = [
                  'hr_cd'=>$value->hr_cd,
                  'user_name'=>$value->user_name,
                ];
            }
        }
        return view(
            'projecthr.edit',
            ['project' => $project, 'details' => $pd, 'hrs' => $hrs, 'roles' => $roles]
        );
    }

    public function update(ProjectHrRequest $request, $project_no)
    {
        $user = Auth::user();
        if (!$user->is_admin) {
            return redirect(route('home'))->with('not_admin', '1');
        }

        $roles = MstRole::all();
        $msg = null;
        $c = 0;
        if (empty($request->selects)) {
          return redirect('/project')->with('no_regist', '1');
        }
        foreach ($request->selects as $select) {
            $checklist = [];
            for ($i = 0; $i < count($roles); $i++) {
                if (isset($select[$roles[$i]->name]['hrs'])) {
                    for ($a = 0; $a < count($select[$roles[$i]->name]['hrs']); $a++) {
                        if (in_array($select[$roles[$i]->name]['hrs'][$a], $checklist)) {
                            if ($msg == null) {
                                $msg = new MessageBag();
                            }
                            $msg->add("selects.{$c}.{$roles[$i]->name}.hrs", "複数の役割を1工程の中で割り当てる事は出来ません");
                        }
                        $checklist[] = $select[$roles[$i]->name]['hrs'][$a];
                    }
                }
            }
            $c++;
        }
        if ($msg != null) {
            return redirect("/projecthr/{$project_no}")->withErrors($msg)->withInput();
        }


        DB::transaction(function () use ($request, $project_no, $roles) {
            TrnProjectDetailHr::where('project_no', $project_no)->delete();
            foreach ($request->selects as $select) {
                for ($i = 0; $i < count($roles); $i++) {
                    if (isset($select[$roles[$i]->name]['hrs'])) {
                        for ($a = 0; $a < count($select[$roles[$i]->name]['hrs']); $a++) {
                            $projectDetailHrs = new TrnProjectDetailHr();
                            $projectDetailHrs->project_no = $project_no;
                            $projectDetailHrs->process_id = $select['process_id'];
                            $projectDetailHrs->hr_cd = $select[$roles[$i]->name]['hrs'][$a];
                            $projectDetailHrs->role_id = $roles[$i]->role_id;
                            $projectDetailHrs->save();
                        }
                    }
                }
            }
        });

        return redirect('/project')->with('registered', '1');
    }
}

class ProjectHrService
{
    public static function getEnableHrs($project_from, $role_id)
    {
        $sql = <<<SQL
select
  u.hr_cd
  , h.user_name
from
  mst_hr_unit_price u
  inner join mst_hr h
    on h.hr_cd = u.hr_cd
where
  u.role_id = {$role_id}
  and u.from_date <= '{$project_from}'
group by
  u.hr_cd
  , h.user_name
SQL;
        $results = DB::select($sql);

        return $results;
    }
}
