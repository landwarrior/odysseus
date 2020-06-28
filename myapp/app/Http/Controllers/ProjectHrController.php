<?php

namespace App\Http\Controllers;

use Illuminate\Http\Request;
use App\MstHr;
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
        $processes = MstProcess::all();
        $roles = MstRole::all();
        $projectDetail = TrnProjectDetail::where('project_no', $project_no)->get();
        $projectDetailHrs = TrnProjectDetailHr::where('project_no', $project_no)->get();
        $pd = [];
        foreach ($projectDetail as $detail) {
            $process_name = '';
            $selected_hr = [];
            foreach ($processes as $process) {
                if ($process->process_id == $detail->process_id) {
                    $process_name = $process->name;
                    break;
                }
            }
            foreach ($projectDetailHrs as $selected) {
                if ($selected->process_id == $detail->process_id) {
                    $selected_hr[$selected->hr_cd] = $selected->role_id;
                }
            }
            array_push(
                $pd, [
                    'process_id' => $detail->process_id,
                    'process_name' => $process_name,
                    'man_per_day' => $detail->man_per_day,
                    'pre_cost' => $detail->pre_cost,
                    'selected' => $selected_hr,
                ]
            );
        }
        $hrs = MstHr::all();
        $project = TrnProject::findOrFail($project_no);
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
