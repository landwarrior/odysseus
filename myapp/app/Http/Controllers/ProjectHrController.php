<?php

namespace App\Http\Controllers;

use Illuminate\Http\Request;
use App\MstHr;
use App\MstProcess;
use App\TrnProject;
use App\TrnProjectDetail;
use App\TrnProjectDetailHr;
use Illuminate\Support\Facades\Auth;
use Illuminate\Support\Facades\DB;

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
        $projectDetail = TrnProjectDetail::where('project_no', $project_no)->get();
        $pd = [];
        foreach ($projectDetail as $detail) {
            $process_name = '';
            foreach ($processes as $process) {
                if ($process->process_id == $detail->process_id) {
                    $process_name = $process->name;
                    break;
                }
            }
            array_push(
                $pd, [
                    'process_id' => $detail->process_id,
                    'process_name' => $process_name,
                    'man_per_day' => $detail->man_per_day,
                    'pre_cost' => $detail->pre_cost,
                ]
            );
        }
        $hrs = MstHr::all();
        $projectDetailHrs = TrnProjectDetailHr::where('project_no', $project_no)->get();
        $project = TrnProject::findOrFail($project_no);
        return view(
            'projecthr.edit',
            ['project' => $project, 'details' => $pd, 'hrs' => $hrs]
        );
    }
}
